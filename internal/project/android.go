package project

import (
	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/parse"
)

type androidAdapter struct{}

// NewAndroidAdapter returns the Android platform adapter.
func NewAndroidAdapter() Adapter { return androidAdapter{} }

func (androidAdapter) Platform() Platform { return Android }

func (androidAdapter) Detect(root string) (bool, ProjectInfo, error) {
	settings, hasSettings := firstExisting(root, "settings.gradle.kts", "settings.gradle")
	build, hasBuild := firstExisting(root, "build.gradle.kts", "build.gradle")
	if !hasSettings || !hasBuild {
		return false, ProjectInfo{}, nil
	}
	return true, ProjectInfo{
		Platform:    Android,
		Root:        root,
		MarkerFiles: []string{settings, build},
	}, nil
}

func (androidAdapter) Files(root string) ([]string, error) {
	return parse.Walk(root, parse.WalkOptions{
		Include: []string{"*.kt", "**/res/layout/*.xml"},
	})
}

// Language is left unimplemented: wiring it to internal/lang/kotlin
// would create project -> lang/kotlin -> extract -> project (extract
// imports project.Platform for CallSite.Platform), the exact cycle
// flagged as a TODO on Adapter in adapter.go. Language selection for
// scanning happens at the point of use (internal/cli) instead, which is
// that TODO's predicted resolution playing out.
func (androidAdapter) Language(file string) *sitter.Language { return nil }
func (androidAdapter) CallQueries() []Query                  { return nil }
func (androidAdapter) InitDetector() Detector                { return nil }
func (androidAdapter) ScreenDetectors() []Detector           { return nil }
func (androidAdapter) EventDetectors() []Detector            { return nil }
func (androidAdapter) Formatter() Formatter                  { return nil }
