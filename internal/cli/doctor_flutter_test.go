package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func TestDoctorFlutterSampleApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/sample-app")
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
}

func TestDoctorFlutterRoutesApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/routes-app")
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
