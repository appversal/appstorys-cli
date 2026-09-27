package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func TestPlanIntegrateAndroidViewsOverlay(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/android/integrate-views-overlay")
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

	var overlay int
	for _, s := range suggestions {
		if s.Kind == "overlay-host" {
			overlay++
			if s.Anchor != "HomeActivity" {
				t.Errorf("overlay-host Anchor = %q, want HomeActivity", s.Anchor)
			}
		}
	}
	if overlay != 1 {
		t.Fatalf("overlay-host suggestions = %d, want 1: %+v", overlay, suggestions)
	}

	// Screen tracking already exists in the fixture, so no screen
	// suggestion should be proposed alongside the overlay-host one.
	for _, s := range suggestions {
		if s.Kind == "screen" {
			t.Errorf("unexpected screen suggestion, HomeActivity is already tracked: %+v", s)
		}
	}
}

// TestPlanIntegrateAndroidViewsOverlayHelperMethod covers an Activity
// whose setContent { ... } lives in a private helper method called from
// onCreate, rather than directly inside onCreate itself — a common
// structuring style that LocateClassAnyMethodBlock has to see past.
func TestPlanIntegrateAndroidViewsOverlayHelperMethod(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/android/integrate-views-overlay-helper")
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

	var overlay int
	for _, s := range suggestions {
		if s.Kind == "overlay-host" {
			overlay++
			if s.Anchor != "HomeActivity" {
				t.Errorf("overlay-host Anchor = %q, want HomeActivity", s.Anchor)
			}
		}
	}
	if overlay != 1 {
		t.Fatalf("overlay-host suggestions = %d, want 1 (setContent found via helper method): %+v", overlay, suggestions)
	}
}
