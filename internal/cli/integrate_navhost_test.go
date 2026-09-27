package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func TestPlanIntegrateAndroidNavHost(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/android/integrate-navhost")
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

	var overlay, screens int
	var screenNames []string
	for _, s := range suggestions {
		switch s.Kind {
		case "overlay-host":
			overlay++
			if s.Anchor != "AppNavHost" {
				t.Errorf("overlay-host suggestion Anchor = %q, want AppNavHost", s.Anchor)
			}
		case "screen":
			screens++
			screenNames = append(screenNames, s.Anchor)
		default:
			t.Errorf("unexpected suggestion kind %q", s.Kind)
		}
	}

	if overlay != 1 {
		t.Errorf("overlay-host suggestions = %d, want 1", overlay)
	}
	// Exactly the two NavHost destinations, not MainActivity itself
	// (it just hosts the NavHost).
	if screens != 2 {
		t.Errorf("screen suggestions = %d, want 2 (route destinations only): %v", screens, screenNames)
	}
	wantRoutes := map[string]bool{"home": true, "order_history": true}
	for _, r := range screenNames {
		if !wantRoutes[r] {
			t.Errorf("unexpected screen suggestion anchor %q", r)
		}
	}
}
