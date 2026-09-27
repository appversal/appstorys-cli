package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/appversal/appstorys-cli/internal/lang/kotlin"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
)

func TestPlanIntegrateAndroidApplyStaysValidKotlin(t *testing.T) {
	src := "../../testdata/fixtures/android/integrate-navhost"
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
	suggestions, err := planIntegrateAndroid(app, sites)
	if err != nil {
		t.Fatalf("planIntegrateAndroid() error = %v", err)
	}
	if len(suggestions) == 0 {
		t.Fatal("no suggestions produced")
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

	// Re-scanning should now find the inserted calls, and re-planning
	// should have nothing left to propose.
	sitesAfter, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() after apply error = %v", err)
	}
	again, err := planIntegrateAndroid(app, sitesAfter)
	if err != nil {
		t.Fatalf("planIntegrateAndroid() after apply error = %v", err)
	}
	if len(again) != 0 {
		t.Errorf("planIntegrateAndroid() after apply = %+v, want none (idempotent)", again)
	}
}
