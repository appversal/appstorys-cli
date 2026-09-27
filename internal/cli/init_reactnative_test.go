package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/appversal/appstorys-cli/internal/lang/ts"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
)

func planReactNativeFor(t *testing.T, root string) []suggestionSummary {
	t.Helper()
	app := testApp(t, root)
	info, ok := mustDetect(t, app, project.ReactNative)
	if !ok {
		t.Fatal("react-native not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}
	suggestions, err := planInitReactNative(app, sites)
	if err != nil {
		t.Fatalf("planInitReactNative() error = %v", err)
	}
	out := make([]suggestionSummary, len(suggestions))
	for i, s := range suggestions {
		out[i] = suggestionSummary{File: s.File, Kind: s.Kind}
	}
	return out
}

func TestPlanInitReactNativeAlreadyDone(t *testing.T) {
	got := planReactNativeFor(t, "../../testdata/fixtures/reactnative/navigator-app")
	if len(got) != 0 {
		t.Errorf("planInitReactNative() = %+v, want no suggestions (already initialized)", got)
	}
}

func TestPlanInitReactNativeExistingUseEffect(t *testing.T) {
	got := planReactNativeFor(t, "../../testdata/fixtures/reactnative/init-existing-useeffect")
	if len(got) != 1 || got[0].File != "src/App.tsx" {
		t.Errorf("planInitReactNative() = %+v, want 1 suggestion targeting src/App.tsx", got)
	}
}

func TestPlanInitReactNativeNoUseEffect(t *testing.T) {
	got := planReactNativeFor(t, "../../testdata/fixtures/reactnative/init-no-useeffect")
	if len(got) != 1 || got[0].File != "src/App.tsx" {
		t.Errorf("planInitReactNative() = %+v, want 1 suggestion targeting src/App.tsx", got)
	}
}

// TestPlanInitReactNativeApplyStaysValidTS applies the generated
// suggestion to scratch copies of both fixtures and re-parses the
// result, failing on a tree-sitter syntax error.
func TestPlanInitReactNativeApplyStaysValidTS(t *testing.T) {
	for _, fixture := range []string{
		"../../testdata/fixtures/reactnative/init-existing-useeffect",
		"../../testdata/fixtures/reactnative/init-no-useeffect",
	} {
		t.Run(fixture, func(t *testing.T) {
			dst := t.TempDir()
			copyDir(t, fixture, dst)

			app := testApp(t, dst)
			info, ok := mustDetect(t, app, project.ReactNative)
			if !ok {
				t.Fatal("react-native not detected")
			}
			sites, err := scanPlatform(app, info)
			if err != nil {
				t.Fatalf("scanPlatform() error = %v", err)
			}
			suggestions, err := planInitReactNative(app, sites)
			if err != nil {
				t.Fatalf("planInitReactNative() error = %v", err)
			}
			if len(suggestions) == 0 {
				t.Fatal("no suggestions produced")
			}
			if err := applySuggestions(dst, suggestions); err != nil {
				t.Fatalf("applySuggestions() error = %v", err)
			}

			pool, err := parse.NewPool(ts.Language(), 1)
			if err != nil {
				t.Fatalf("NewPool() error = %v", err)
			}
			defer pool.Close()

			for _, s := range suggestions {
				data, err := os.ReadFile(filepath.Join(dst, s.File))
				if err != nil {
					t.Fatalf("reading applied %s: %v", s.File, err)
				}
				tree, err := pool.Parse(context.Background(), data)
				if err != nil {
					t.Fatalf("parsing applied %s: %v", s.File, err)
				}
				if hasSyntaxError(tree.RootNode()) {
					t.Errorf("applied %s has a syntax error:\n%s", s.File, data)
				}
				tree.Close()
			}

			sitesAfter, err := scanPlatform(app, info)
			if err != nil {
				t.Fatalf("scanPlatform() after apply error = %v", err)
			}
			again, err := planInitReactNative(app, sitesAfter)
			if err != nil {
				t.Fatalf("planInitReactNative() after apply error = %v", err)
			}
			if len(again) != 0 {
				t.Errorf("planInitReactNative() after apply = %+v, want none (idempotent)", again)
			}
		})
	}
}
