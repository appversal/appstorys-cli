package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func TestDoctorFlutterGoRouterApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/gorouter-app")
	info, ok := mustDetect(t, app, project.Flutter)
	if !ok {
		t.Fatal("flutter not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}

	results, err := doctorFlutter(app, sites)
	if err != nil {
		t.Fatalf("doctorFlutter() error = %v", err)
	}

	init := findResult(results, "init", "-")
	if init == nil || init.Status != "pass" {
		t.Errorf("init check = %+v, want pass", init)
	}
	home := findResult(results, "screen", "HomeScreen")
	if home == nil || home.Status != "pass" || home.TrackedAs != "Home Screen" || home.Overlay != "root" {
		t.Errorf("HomeScreen check = %+v, want pass/Home Screen/root", home)
	}
	settings := findResult(results, "screen", "SettingsScreen")
	if settings == nil || settings.Status != "fail: not tracked" {
		t.Errorf("SettingsScreen check = %+v, want fail: not tracked", settings)
	}
}

func TestPlanIntegrateFlutterGoRouterApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/gorouter-app")
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
	if len(suggestions) != 1 || suggestions[0].Kind != "screen" || suggestions[0].File != "lib/settings_screen.dart" {
		t.Fatalf("suggestions = %+v, want exactly 1 screen suggestion for lib/settings_screen.dart", suggestions)
	}
}
