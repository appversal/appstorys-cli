package project

import (
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"
	"gopkg.in/yaml.v3"

	"github.com/appversal/appstorys-cli/internal/parse"
)

type flutterAdapter struct{}

// NewFlutterAdapter returns the Flutter platform adapter.
func NewFlutterAdapter() Adapter { return flutterAdapter{} }

func (flutterAdapter) Platform() Platform { return Flutter }

type pubspecFile struct {
	Environment map[string]any `yaml:"environment"`
	Flutter     any            `yaml:"flutter"`
}

func (flutterAdapter) Detect(root string) (bool, ProjectInfo, error) {
	const rel = "pubspec.yaml"
	if !fileExists(root, rel) {
		return false, ProjectInfo{}, nil
	}
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return false, ProjectInfo{}, err
	}
	var doc pubspecFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		// Not valid YAML: treat as "not a Flutter project" rather than
		// a hard detection error, matching Detect's contract of
		// returning false for a non-match.
		return false, ProjectInfo{}, nil
	}

	isFlutter := doc.Flutter != nil
	if v, ok := doc.Environment["flutter"]; ok && v != nil {
		isFlutter = true
	}
	if !isFlutter {
		return false, ProjectInfo{}, nil
	}
	return true, ProjectInfo{
		Platform:    Flutter,
		Root:        root,
		MarkerFiles: []string{rel},
	}, nil
}

func (flutterAdapter) Files(root string) ([]string, error) {
	return parse.Walk(root, parse.WalkOptions{
		Include: []string{"*.dart"},
	})
}

func (flutterAdapter) Language(file string) *sitter.Language { return nil }
func (flutterAdapter) CallQueries() []Query                  { return nil }
func (flutterAdapter) InitDetector() Detector                { return nil }
func (flutterAdapter) ScreenDetectors() []Detector           { return nil }
func (flutterAdapter) EventDetectors() []Detector            { return nil }
func (flutterAdapter) Formatter() Formatter                  { return nil }
