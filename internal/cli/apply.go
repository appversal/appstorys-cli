package cli

import (
	"os"
	"path/filepath"
	"sort"

	"github.com/appversal/appstorys-cli/internal/patch"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// applySuggestions writes every suggestion to disk. Suggestions
// targeting the same existing file are applied highest-InsertAt-first:
// an insertion never shifts the byte offsets of anything earlier in the
// same file, but it does shift everything after it, so applying in
// generation order (typically ascending file position, since detectors
// walk top to bottom) silently invalidates not-yet-applied offsets and
// corrupts later insertions. New-file suggestions have no such conflict
// and are applied in any order.
func applySuggestions(root string, suggestions []suggest.Suggestion) error {
	byFile := make(map[string][]suggest.Suggestion, len(suggestions))
	var order []string
	for _, s := range suggestions {
		if _, ok := byFile[s.File]; !ok {
			order = append(order, s.File)
		}
		byFile[s.File] = append(byFile[s.File], s)
	}

	for _, file := range order {
		group := byFile[file]
		sort.SliceStable(group, func(i, j int) bool { return group[i].InsertAt > group[j].InsertAt })
		for _, s := range group {
			_, exists := readTarget(root, s.File)
			if err := patch.Apply(root, toEdit(s, !exists)); err != nil {
				return err
			}
		}
	}
	return nil
}

// readTarget reads a suggestion's target file, returning exists=false
// (not an error) if it doesn't exist yet — the common case for a
// NewFile-shaped suggestion.
func readTarget(root, file string) (data []byte, exists bool) {
	data, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		return nil, false
	}
	return data, true
}

func toEdit(s suggest.Suggestion, newFile bool) patch.Edit {
	return patch.Edit{
		File:     s.File,
		NewFile:  newFile,
		InsertAt: s.InsertAt,
		Snippet:  s.Snippet,
		Reason:   s.Reason,
	}
}
