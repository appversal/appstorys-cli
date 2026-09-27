// Package patch turns a Suggestion into a file change: rendering it as
// a diff, or writing it to disk. Every edit this phase produces is a
// pure insertion — no deletions or replacements — so Render builds a
// unified-diff hunk directly from the insertion point rather than
// running a general diff algorithm.
package patch

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Edit is one proposed change to a single file.
type Edit struct {
	File     string // root-relative path
	NewFile  bool   // true if File doesn't exist yet; Snippet is the whole new content
	InsertAt int    // byte offset into the current file content; ignored if NewFile
	Snippet  string
	Reason   string // human-readable summary, shown above the diff
}

// Insert returns original with snippet inserted at byte offset at.
func Insert(original []byte, at int, snippet string) []byte {
	at = max(0, min(at, len(original)))
	out := make([]byte, 0, len(original)+len(snippet))
	out = append(out, original[:at]...)
	out = append(out, snippet...)
	out = append(out, original[at:]...)
	return out
}

// Render produces a unified-diff-style hunk for edit. original is the
// file's current content; pass nil for a NewFile edit.
//
// An insertion doesn't have to land at a line boundary (the manifest
// android:name attribute and an init call inserted after a function's
// "{" both land mid-line), so the line containing InsertAt is rendered
// as changed — its old text removed, its new (prefix+snippet+suffix)
// text added — rather than assumed to already sit on its own line.
func Render(edit Edit, original []byte) string {
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n", edit.File)
	fmt.Fprintf(&b, "+++ b/%s\n", edit.File)

	if edit.NewFile {
		lines := strings.Split(strings.TrimRight(edit.Snippet, "\n"), "\n")
		fmt.Fprintf(&b, "@@ -0,0 +1,%d @@\n", len(lines))
		for _, l := range lines {
			fmt.Fprintf(&b, "+%s\n", l)
		}
		return b.String()
	}

	at := max(0, min(edit.InsertAt, len(original)))

	lineStart := bytes.LastIndexByte(original[:at], '\n') + 1 // 0 if no preceding newline
	lineEnd := len(original)
	if rel := bytes.IndexByte(original[at:], '\n'); rel != -1 {
		lineEnd = at + rel
	}
	prefix := string(original[lineStart:at])
	suffix := string(original[at:lineEnd])
	originalLine := string(original[lineStart:lineEnd])
	lineNo := 1 + bytes.Count(original[:lineStart], []byte("\n"))

	snippetLines := strings.Split(edit.Snippet, "\n")
	newLines := make([]string, 0, len(snippetLines))
	if len(snippetLines) == 1 {
		newLines = append(newLines, prefix+snippetLines[0]+suffix)
	} else {
		newLines = append(newLines, prefix+snippetLines[0])
		newLines = append(newLines, snippetLines[1:len(snippetLines)-1]...)
		newLines = append(newLines, snippetLines[len(snippetLines)-1]+suffix)
	}
	// A snippet ending in "\n" (the common case) leaves a wholly-empty
	// trailing element once suffix is empty too; drop it.
	if len(newLines) > 1 && newLines[len(newLines)-1] == "" {
		newLines = newLines[:len(newLines)-1]
	}

	const context = 2
	origLines := strings.Split(string(original), "\n")
	idx := lineNo - 1
	startCtx := max(0, idx-context)
	endCtx := min(len(origLines), idx+1+context)
	origCount := endCtx - startCtx
	newCount := origCount - 1 + len(newLines)

	fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", startCtx+1, origCount, startCtx+1, newCount)
	for i := startCtx; i < idx; i++ {
		fmt.Fprintf(&b, " %s\n", origLines[i])
	}
	fmt.Fprintf(&b, "-%s\n", originalLine)
	for _, l := range newLines {
		fmt.Fprintf(&b, "+%s\n", l)
	}
	for i := idx + 1; i < endCtx; i++ {
		fmt.Fprintf(&b, " %s\n", origLines[i])
	}
	return b.String()
}

// Apply writes edit's result to root/edit.File.
func Apply(root string, edit Edit) error {
	path := filepath.Join(root, edit.File)
	if edit.NewFile {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("patch: creating %s: %w", filepath.Dir(edit.File), err)
		}
		if err := os.WriteFile(path, []byte(edit.Snippet), 0o644); err != nil {
			return fmt.Errorf("patch: writing %s: %w", edit.File, err)
		}
		return nil
	}

	original, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("patch: reading %s: %w", edit.File, err)
	}
	updated := Insert(original, edit.InsertAt, edit.Snippet)
	if err := os.WriteFile(path, updated, 0o644); err != nil {
		return fmt.Errorf("patch: writing %s: %w", edit.File, err)
	}
	return nil
}

// GitClean reports whether root's git working tree has no uncommitted
// changes. --apply requires this so a bad suggestion is always
// trivially revertable with git.
func GitClean(root string) (bool, error) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("patch: git status: %w", err)
	}
	return len(strings.TrimSpace(string(out))) == 0, nil
}
