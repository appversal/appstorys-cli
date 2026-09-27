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

func TestPlanInitFlutterAlreadyDone(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/sample-app")
	info, ok := mustDetect(t, app, project.Flutter)
	if !ok {
		t.Fatal("flutter not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}
	suggestions, err := planInitFlutter(app, sites)
	if err != nil {
		t.Fatalf("planInitFlutter() error = %v", err)
	}
	if len(suggestions) != 0 {
		t.Errorf("planInitFlutter() = %+v, want no suggestions (already initialized)", suggestions)
	}
}

func TestPlanInitFlutterMissing(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/init-missing")
	info, ok := mustDetect(t, app, project.Flutter)
	if !ok {
		t.Fatal("flutter not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}
	suggestions, err := planInitFlutter(app, sites)
	if err != nil {
		t.Fatalf("planInitFlutter() error = %v", err)
	}
	if len(suggestions) != 1 || suggestions[0].File != "lib/main.dart" {
		t.Fatalf("planInitFlutter() = %+v, want 1 suggestion targeting lib/main.dart", suggestions)
	}
}

// TestPlanInitFlutterApplyStaysValid applies the suggestion to a
// scratch copy, re-parses for syntax errors, and confirms doctor then
// reports the init check as passing.
func TestPlanInitFlutterApplyStaysValid(t *testing.T) {
	src := "../../testdata/fixtures/flutter/init-missing"
	dst := t.TempDir()
	copyDir(t, src, dst)

	app := testApp(t, dst)
	info, ok := mustDetect(t, app, project.Flutter)
	if !ok {
		t.Fatal("flutter not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}
	suggestions, err := planInitFlutter(app, sites)
	if err != nil {
		t.Fatalf("planInitFlutter() error = %v", err)
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
	again, err := planInitFlutter(app, sitesAfter)
	if err != nil {
		t.Fatalf("planInitFlutter() after apply error = %v", err)
	}
	if len(again) != 0 {
		t.Errorf("planInitFlutter() after apply = %+v, want none (idempotent)", again)
	}

	results, err := doctorFlutter(app, sitesAfter)
	if err != nil {
		t.Fatalf("doctorFlutter() after apply error = %v", err)
	}
	init := findResult(results, "init", "-")
	if init == nil || init.Status != "pass" {
		t.Errorf("init check after apply = %+v, want pass", init)
	}
}
