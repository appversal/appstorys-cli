package project

import (
	"encoding/json"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/parse"
)

type reactNativeAdapter struct{}

// NewReactNativeAdapter returns the React Native platform adapter.
func NewReactNativeAdapter() Adapter { return reactNativeAdapter{} }

func (reactNativeAdapter) Platform() Platform { return ReactNative }

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func (reactNativeAdapter) Detect(root string) (bool, ProjectInfo, error) {
	const rel = "package.json"
	if !fileExists(root, rel) {
		return false, ProjectInfo{}, nil
	}
	data, err := os.ReadFile(filepath.Join(root, rel))
	if err != nil {
		return false, ProjectInfo{}, err
	}
	var doc packageJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		return false, ProjectInfo{}, nil
	}
	_, inDeps := doc.Dependencies["react-native"]
	_, inDevDeps := doc.DevDependencies["react-native"]
	if !inDeps && !inDevDeps {
		return false, ProjectInfo{}, nil
	}
	return true, ProjectInfo{
		Platform:    ReactNative,
		Root:        root,
		MarkerFiles: []string{rel},
	}, nil
}

func (reactNativeAdapter) Files(root string) ([]string, error) {
	return parse.Walk(root, parse.WalkOptions{
		Include: []string{"*.ts", "*.tsx", "*.js", "*.jsx"},
	})
}

func (reactNativeAdapter) Language(file string) *sitter.Language { return nil }
func (reactNativeAdapter) CallQueries() []Query                  { return nil }
func (reactNativeAdapter) InitDetector() Detector                { return nil }
func (reactNativeAdapter) ScreenDetectors() []Detector           { return nil }
func (reactNativeAdapter) EventDetectors() []Detector            { return nil }
func (reactNativeAdapter) Formatter() Formatter                  { return nil }
