package cli

import (
	"strings"
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func findResult(results []DoctorResult, check, screen string) *DoctorResult {
	for i := range results {
		if results[i].Check == check && results[i].Screen == screen {
			return &results[i]
		}
	}
	return nil
}

func TestDoctorAndroidClean(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/android/sample-app")
	info, ok := mustDetect(t, app, project.Android)
	if !ok {
		t.Fatal("android not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}

	results, err := doctorAndroid(app, sites)
	if err != nil {
		t.Fatalf("doctorAndroid() error = %v", err)
	}

	init := findResult(results, "init", "-")
	if init == nil || init.Status != "pass" {
		t.Errorf("init check = %+v, want pass", init)
	}

	home := findResult(results, "screen", "HomeActivity")
	if home == nil || home.Status != "pass" || home.TrackedAs != "Home Screen" || home.Overlay != "root" {
		t.Errorf("HomeActivity screen check = %+v, want pass/Home Screen/root", home)
	}

	settings := findResult(results, "screen", "SettingsActivity")
	if settings == nil || settings.Status != "pass" || settings.TrackedAs != "Settings Screen" {
		t.Errorf("SettingsActivity screen check = %+v, want pass/Settings Screen", settings)
	}
}

func TestDoctorAndroidBroken(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/android/doctor-broken")
	info, ok := mustDetect(t, app, project.Android)
	if !ok {
		t.Fatal("android not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}

	results, err := doctorAndroid(app, sites)
	if err != nil {
		t.Fatalf("doctorAndroid() error = %v", err)
	}

	init := findResult(results, "init", "-")
	if init == nil || !strings.HasPrefix(init.Status, "fail:") || !strings.Contains(init.Status, "BrokenApp") {
		t.Errorf("init check = %+v, want a fail mentioning BrokenApp", init)
	}

	home := findResult(results, "screen", "HomeActivity")
	if home == nil || home.Status != "fail: no overlay host" {
		t.Errorf("HomeActivity screen check = %+v, want fail: no overlay host", home)
	}

	settings := findResult(results, "screen", "SettingsActivity")
	if settings == nil || settings.Status != "fail: not tracked" {
		t.Errorf("SettingsActivity screen check = %+v, want fail: not tracked", settings)
	}
}

func mustDetect(t *testing.T, app *App, platform project.Platform) (project.ProjectInfo, bool) {
	t.Helper()
	infos, err := project.Detect(app.Root, []project.Platform{platform})
	if err != nil {
		t.Fatalf("Detect() error = %v", err)
	}
	if len(infos) == 0 {
		return project.ProjectInfo{}, false
	}
	return infos[0], true
}
