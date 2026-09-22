package parse

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// defaultExcludeDirs are directory basenames Walk always skips, matching
// the spec's ignores list (.gitignore, build/, Pods/, node_modules/,
// .dart_tool/, generated files, plus config overrides).
var defaultExcludeDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"build":        true,
	"Pods":         true,
	".dart_tool":   true,
	"generated":    true,
}

// WalkOptions filters the files Walk returns.
type WalkOptions struct {
	// Include, if non-empty, restricts results to files matching at
	// least one pattern. A pattern with no "/" matches by basename
	// anywhere in the tree (e.g. "*.kt"); a pattern containing "/" is
	// matched against the root-relative path and may use "**" to match
	// zero or more path segments (e.g. "app/src/**/*.kt").
	Include []string
	// Exclude adds extra patterns, in the same syntax as Include,
	// merged with the default ignored directories and the root's
	// .gitignore (if present; negation patterns in .gitignore are not
	// supported and are skipped).
	Exclude []string
}

// Walk returns root-relative, slash-separated paths of every file under
// root that Include/Exclude admit, skipping default-ignored directories
// entirely rather than descending into them.
func Walk(root string, opts WalkOptions) ([]string, error) {
	excludes := append([]string{}, opts.Exclude...)
	excludes = append(excludes, gitignorePatterns(root)...)

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		base := d.Name()

		if d.IsDir() {
			if defaultExcludeDirs[base] || matchesAny(excludes, rel, base) {
				return filepath.SkipDir
			}
			return nil
		}

		if matchesAny(excludes, rel, base) {
			return nil
		}
		if len(opts.Include) > 0 && !matchesAny(opts.Include, rel, base) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func gitignorePatterns(root string) []string {
	data, err := os.ReadFile(filepath.Join(root, ".gitignore"))
	if err != nil {
		return nil
	}
	var patterns []string
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") {
			continue
		}
		patterns = append(patterns, strings.TrimPrefix(strings.TrimSuffix(line, "/"), "/"))
	}
	return patterns
}

func matchesAny(patterns []string, relPath, base string) bool {
	for _, p := range patterns {
		if matchPattern(p, relPath, base) {
			return true
		}
	}
	return false
}

func matchPattern(pattern, relPath, base string) bool {
	if !strings.Contains(pattern, "/") {
		ok, _ := filepath.Match(pattern, base)
		return ok
	}
	return matchSegments(strings.Split(pattern, "/"), strings.Split(relPath, "/"))
}

// matchSegments matches a "/"-split glob pattern against a "/"-split
// path, where a "**" segment matches zero or more path segments and
// every other segment is matched with filepath.Match.
func matchSegments(pattern, path []string) bool {
	if len(pattern) == 0 {
		return len(path) == 0
	}
	if pattern[0] == "**" {
		if matchSegments(pattern[1:], path) {
			return true
		}
		if len(path) == 0 {
			return false
		}
		return matchSegments(pattern, path[1:])
	}
	if len(path) == 0 {
		return false
	}
	if ok, err := filepath.Match(pattern[0], path[0]); err != nil || !ok {
		return false
	}
	return matchSegments(pattern[1:], path[1:])
}
