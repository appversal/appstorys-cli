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

func TestPlanIntegrateReactNative(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/reactnative/navigator-app")
	info, ok := mustDetect(t, app, project.ReactNative)
	if !ok {
		t.Fatal("react-native not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}

	suggestions, err := planIntegrateReactNative(app, sites)
	if err != nil {
		t.Fatalf("planIntegrateReactNative() error = %v", err)
	}

	// One wrap = two edits (open + close tag) sharing one ID.
	if len(suggestions) != 2 {
		t.Fatalf("suggestions = %d, want 2 (paired open/close edit): %+v", len(suggestions), suggestions)
	}
	if suggestions[0].ID != suggestions[1].ID {
		t.Errorf("paired suggestions have different IDs: %q vs %q", suggestions[0].ID, suggestions[1].ID)
	}
	for _, s := range suggestions {
		if s.File != "src/screens/SettingsScreen.tsx" {
			t.Errorf("suggestion File = %q, want src/screens/SettingsScreen.tsx", s.File)
		}
		if s.Anchor != "SettingsScreen" {
			t.Errorf("suggestion Anchor = %q, want SettingsScreen", s.Anchor)
		}
	}

	// HomeScreen is already tracked, so it must not get a suggestion.
	for _, s := range suggestions {
		if s.Anchor == "HomeScreen" {
			t.Errorf("unexpected suggestion for already-tracked HomeScreen: %+v", s)
		}
	}
}

// TestPlanIntegrateReactNativeApplyStaysValid applies the paired wrap
// suggestion to a scratch copy of each fixture, re-parses the result
// for syntax errors, and confirms scanning again finds the screen
// tracked (so the wrapper actually landed correctly, not just "didn't
// crash the parser"). navigator-app covers the JSX route form
// (<Stack.Screen component={Y} />), static-navigator-app the static-API
// form (createXNavigator({ screens: { X: Y } })).
func TestPlanIntegrateReactNativeApplyStaysValid(t *testing.T) {
	for _, fixture := range []string{
		"../../testdata/fixtures/reactnative/navigator-app",
		"../../testdata/fixtures/reactnative/static-navigator-app",
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
			suggestions, err := planIntegrateReactNative(app, sites)
			if err != nil {
				t.Fatalf("planIntegrateReactNative() error = %v", err)
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

			touched := map[string]bool{}
			for _, s := range suggestions {
				touched[s.File] = true
			}
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

			sitesAfter, err := scanPlatform(app, info)
			if err != nil {
				t.Fatalf("scanPlatform() after apply error = %v", err)
			}
			again, err := planIntegrateReactNative(app, sitesAfter)
			if err != nil {
				t.Fatalf("planIntegrateReactNative() after apply error = %v", err)
			}
			if len(again) != 0 {
				t.Errorf("planIntegrateReactNative() after apply = %+v, want none (idempotent)", again)
			}

			results, err := doctorReactNative(app, sitesAfter)
			if err != nil {
				t.Fatalf("doctorReactNative() after apply error = %v", err)
			}
			settings := findResult(results, "screen", "Settings")
			if settings == nil || settings.Status != "pass" {
				t.Errorf("Settings screen check after apply = %+v, want pass", settings)
			}
		})
	}
}
