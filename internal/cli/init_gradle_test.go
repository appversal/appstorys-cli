package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/lang/kotlin"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
)

func TestPlanGradleAndroid(t *testing.T) {
	got := planFor(t, "../../testdata/fixtures/android/init-missing-gradle")

	want := map[string]int{
		"settings.gradle.kts":  1, // jitpack repo
		"app/build.gradle.kts": 3, // dependency + buildFeatures + buildConfigField
	}
	counts := map[string]int{}
	for _, s := range got {
		counts[s.File]++
	}
	for file, n := range want {
		if counts[file] != n {
			t.Errorf("suggestions for %s = %d, want %d (all: %+v)", file, counts[file], n, got)
		}
	}
}

// TestPlanGradleAndroidApplyStaysValidKotlin applies every generated
// suggestion to a scratch copy of the fixture and re-parses each
// touched .gradle.kts file, failing if tree-sitter reports a syntax
// error — the strongest available signal that the inserted Gradle DSL
// text is actually well-formed, short of invoking a real Gradle build.
func TestPlanGradleAndroidApplyStaysValidKotlin(t *testing.T) {
	src := "../../testdata/fixtures/android/init-missing-gradle"
	dst := t.TempDir()
	copyDir(t, src, dst)

	app := testApp(t, dst)
	info, ok := mustDetect(t, app, project.Android)
	if !ok {
		t.Fatal("android not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}
	classes, err := allKotlinClasses(app)
	if err != nil {
		t.Fatalf("allKotlinClasses() error = %v", err)
	}
	suggestions, err := planInitAndroid(app, sites, classes)
	if err != nil {
		t.Fatalf("planInitAndroid() error = %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("planInitAndroid() returned no suggestions, expected the Gradle wiring set")
	}

	touched := map[string]bool{}
	for _, s := range suggestions {
		touched[s.File] = true
	}
	if err := applySuggestions(dst, suggestions); err != nil {
		t.Fatalf("applySuggestions() error = %v", err)
	}

	pool, err := parse.NewPool(kotlin.Language(), 1)
	if err != nil {
		t.Fatalf("NewPool() error = %v", err)
	}
	defer pool.Close()

	for file := range touched {
		data, err := os.ReadFile(filepath.Join(dst, file))
		if err != nil {
			t.Fatalf("reading applied %s: %v", file, err)
		}
		tree, err := pool.Parse(context.Background(), data)
		if err != nil {
			t.Fatalf("parsing applied %s: %v", file, err)
		}
		if hasSyntaxError(tree.RootNode()) {
			t.Errorf("applied %s has a syntax error:\n%s", file, data)
		}
		tree.Close()
	}
}

func hasSyntaxError(n *sitter.Node) bool {
	if n.IsError() {
		return true
	}
	for i := range n.ChildCount() {
		if hasSyntaxError(n.Child(i)) {
			return true
		}
	}
	return false
}

func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copying fixture: %v", err)
	}
}
