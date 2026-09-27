package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func TestDoctorReactNativeStaticNavigatorApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/reactnative/static-navigator-app")
	info, ok := mustDetect(t, app, project.ReactNative)
	if !ok {
		t.Fatal("react-native not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}

	results, err := doctorReactNative(app, sites)
	if err != nil {
		t.Fatalf("doctorReactNative() error = %v", err)
	}

	init := findResult(results, "init", "-")
	if init == nil || init.Status != "pass" {
		t.Errorf("init check = %+v, want pass", init)
	}
	home := findResult(results, "screen", "Home")
	if home == nil || home.Status != "pass" || home.TrackedAs != "Home Screen" || home.Overlay != "root" {
		t.Errorf("Home screen check = %+v, want pass/Home Screen/root", home)
	}
	settings := findResult(results, "screen", "Settings")
	if settings == nil || settings.Status != "fail: not tracked" {
		t.Errorf("Settings screen check = %+v, want fail: not tracked", settings)
	}
}

func TestPlanIntegrateReactNativeStaticNavigatorApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/reactnative/static-navigator-app")
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
	if len(suggestions) != 2 {
		t.Fatalf("suggestions = %d, want 2 (paired open/close edit): %+v", len(suggestions), suggestions)
	}
	if suggestions[0].ID != suggestions[1].ID {
		t.Errorf("paired suggestions have different IDs: %q vs %q", suggestions[0].ID, suggestions[1].ID)
	}
	for _, s := range suggestions {
		if s.File != "src/screens/SettingsScreen.tsx" || s.Anchor != "SettingsScreen" {
			t.Errorf("suggestion = %+v, want SettingsScreen in src/screens/SettingsScreen.tsx", s)
		}
	}
}
