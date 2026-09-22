package project

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/parse"
)

type iosAdapter struct{}

// NewIOSAdapter returns the iOS platform adapter.
func NewIOSAdapter() Adapter { return iosAdapter{} }

func (iosAdapter) Platform() Platform { return IOS }

func (iosAdapter) Detect(root string) (bool, ProjectInfo, error) {
	if fileExists(root, "Package.swift") {
		return true, ProjectInfo{
			Platform:    IOS,
			Root:        root,
			MarkerFiles: []string{"Package.swift"},
		}, nil
	}
	matches, err := globOneLevelDown(root, "project.pbxproj")
	if err != nil {
		return false, ProjectInfo{}, err
	}
	// globOneLevelDown looks inside every immediate subdirectory; only
	// keep matches that came from a *.xcodeproj directory.
	var pbxproj []string
	for _, m := range matches {
		dir := m[:len(m)-len("/project.pbxproj")]
		if hasSuffixDotXcodeproj(dir) {
			pbxproj = append(pbxproj, m)
		}
	}
	if len(pbxproj) == 0 {
		return false, ProjectInfo{}, nil
	}
	return true, ProjectInfo{
		Platform:    IOS,
		Root:        root,
		MarkerFiles: pbxproj,
	}, nil
}

func hasSuffixDotXcodeproj(dir string) bool {
	const suffix = ".xcodeproj"
	return len(dir) > len(suffix) && dir[len(dir)-len(suffix):] == suffix
}

func (iosAdapter) Files(root string) ([]string, error) {
	return parse.Walk(root, parse.WalkOptions{
		Include: []string{"*.swift"},
	})
}

func (iosAdapter) Language(file string) *sitter.Language { return nil }
func (iosAdapter) CallQueries() []Query                  { return nil }
func (iosAdapter) InitDetector() Detector                { return nil }
func (iosAdapter) ScreenDetectors() []Detector           { return nil }
func (iosAdapter) EventDetectors() []Detector            { return nil }
func (iosAdapter) Formatter() Formatter                  { return nil }
