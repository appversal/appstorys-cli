package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/appversal/appstorys-cli/internal/lang/dart"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
)

func TestPlanIntegrateFlutterRoutesApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/routes-app")
	info, ok := mustDetect(t, app, project.Flutter)
	if !ok {
		t.Fatal("flutter not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}

	suggestions, err := planIntegrateFlutter(app, sites)
	if err != nil {
		t.Fatalf("planIntegrateFlutter() error = %v", err)
	}

	// SettingsScreen is untracked (screen suggestion) but has no Stack
	// at all, so no overlay-host suggestion should be generated for it
	// ("diff only, never auto-apply" — skipped entirely, not guessed).
	if len(suggestions) != 1 {
		t.Fatalf("suggestions = %+v, want exactly 1 (screen tracking only)", suggestions)
	}
	s := suggestions[0]
	if s.Kind != "screen" || s.File != "lib/settings_screen.dart" {
		t.Errorf("suggestion = %+v, want a screen suggestion for lib/settings_screen.dart", s)
	}

	// HomeScreen is already fully tracked; must not get a suggestion.
	for _, s := range suggestions {
		if s.Anchor == "_HomeScreenState" {
			t.Errorf("unexpected suggestion for already-complete HomeScreen: %+v", s)
		}
	}
}

func TestPlanIntegrateFlutterOverlayHost(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/integrate-overlay")
	info, ok := mustDetect(t, app, project.Flutter)
	if !ok {
		t.Fatal("flutter not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}

	suggestions, err := planIntegrateFlutter(app, sites)
	if err != nil {
		t.Fatalf("planIntegrateFlutter() error = %v", err)
	}
	if len(suggestions) != 1 || suggestions[0].Kind != "overlay-host" {
		t.Fatalf("suggestions = %+v, want exactly 1 overlay-host suggestion", suggestions)
	}
}

// TestPlanIntegrateFlutterApplyStaysValid applies both the screen-
// tracking and the overlay-host suggestion (from two different
// fixtures) to scratch copies, re-parses for syntax errors, and
// confirms doctor then reports pass and re-planning finds nothing left.
func TestPlanIntegrateFlutterApplyStaysValid(t *testing.T) {
	for _, fixture := range []string{
		"../../testdata/fixtures/flutter/routes-app",
		"../../testdata/fixtures/flutter/integrate-overlay",
		"../../testdata/fixtures/flutter/gorouter-app",
	} {
		t.Run(fixture, func(t *testing.T) {
			dst := t.TempDir()
			copyDir(t, fixture, dst)

			app := testApp(t, dst)
			info, ok := mustDetect(t, app, project.Flutter)
			if !ok {
				t.Fatal("flutter not detected")
			}
			sites, err := scanPlatform(app, info)
			if err != nil {
				t.Fatalf("scanPlatform() error = %v", err)
			}
			suggestions, err := planIntegrateFlutter(app, sites)
			if err != nil {
				t.Fatalf("planIntegrateFlutter() error = %v", err)
			}
			if len(suggestions) == 0 {
				t.Fatal("no suggestions produced")
			}
			if err := applySuggestions(dst, suggestions); err != nil {
				t.Fatalf("applySuggestions() error = %v", err)
			}

			pool, err := parse.NewPool(dart.Language(), 1)
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
			again, err := planIntegrateFlutter(app, sitesAfter)
			if err != nil {
				t.Fatalf("planIntegrateFlutter() after apply error = %v", err)
			}
			if len(again) != 0 {
				t.Errorf("planIntegrateFlutter() after apply = %+v, want none (idempotent)", again)
			}
		})
	}
}
