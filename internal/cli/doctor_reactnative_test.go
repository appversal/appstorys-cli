package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func TestDoctorReactNativeSampleApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/reactnative/sample-app")
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
}

func TestDoctorReactNativeNavigatorApp(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/reactnative/navigator-app")
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
