package project

import (
	"os"
	"path/filepath"
)

// fileExists reports whether root/rel exists and is a regular file.
func fileExists(root, rel string) bool {
	info, err := os.Stat(filepath.Join(root, rel))
	return err == nil && !info.IsDir()
}

// firstExisting returns the first of rels that exists under root, and
// true, or "" and false if none exist.
func firstExisting(root string, rels ...string) (string, bool) {
	for _, rel := range rels {
		if fileExists(root, rel) {
			return rel, true
		}
	}
	return "", false
}

// globOneLevelDown returns root-relative paths matching pattern within
// the immediate subdirectories of root (not root itself), e.g. to find
// "*.xcodeproj/project.pbxproj" one level under root.
func globOneLevelDown(root, pattern string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	var matches []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		found, err := filepath.Glob(filepath.Join(root, e.Name(), pattern))
		if err != nil {
			return nil, err
		}
		for _, f := range found {
			rel, err := filepath.Rel(root, f)
			if err != nil {
				return nil, err
			}
			matches = append(matches, rel)
		}
	}
	return matches, nil
}
