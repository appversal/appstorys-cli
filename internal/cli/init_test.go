package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/project"
)

func planFor(t *testing.T, root string) []suggestionSummary {
	t.Helper()
	app := testApp(t, root)
	info, ok := mustDetect(t, app, project.Android)
	if !ok {
		t.Fatal("android not detected")
	}
	sites, err := scanPlatform(app, info)
	if err != nil {
		t.Fatalf("scanPlatform() error = %v", err)
	}
	classes, err := allKotlinClasses(app)
	if err != nil {
		t.Fatalf("allKotlinClasses() error = %v", err)
	}
	suggestions, err := planInitAndroid(app, sites, classes)
	if err != nil {
		t.Fatalf("planInitAndroid() error = %v", err)
	}
	out := make([]suggestionSummary, len(suggestions))
	for i, s := range suggestions {
		out[i] = suggestionSummary{File: s.File, Kind: s.Kind}
	}
	return out
}

type suggestionSummary struct {
	File string
	Kind string
}

func TestPlanInitAndroidAlreadyDone(t *testing.T) {
	got := planFor(t, "../../testdata/fixtures/android/sample-app")
	if len(got) != 0 {
		t.Errorf("planInitAndroid() = %+v, want no suggestions (already initialized)", got)
	}
}

func TestPlanInitAndroidMisplacedInitIsLeftAlone(t *testing.T) {
	// doctor-broken has an init call (in the wrong place), which the
	// planner deliberately doesn't try to move/delete (patch is
	// insertion-only) — it should propose nothing.
	got := planFor(t, "../../testdata/fixtures/android/doctor-broken")
	if len(got) != 0 {
		t.Errorf("planInitAndroid() = %+v, want no suggestions (won't move an existing call)", got)
	}
}

func TestPlanInitAndroidNoApplicationClass(t *testing.T) {
	got := planFor(t, "../../testdata/fixtures/android/init-missing-app")
	if len(got) != 2 {
		t.Fatalf("planInitAndroid() = %+v, want 2 suggestions (new App.kt + manifest registration)", got)
	}
	if got[0].File != "app/src/main/java/com/example/fresh/App.kt" {
		t.Errorf("suggestions[0].File = %q, want the new App.kt path", got[0].File)
	}
	if got[1].File != "app/src/main/AndroidManifest.xml" {
		t.Errorf("suggestions[1].File = %q, want AndroidManifest.xml", got[1].File)
	}
}

func TestPlanInitAndroidMissingCallOnly(t *testing.T) {
	got := planFor(t, "../../testdata/fixtures/android/init-missing-call")
	if len(got) != 1 {
		t.Fatalf("planInitAndroid() = %+v, want 1 suggestion (just the init call)", got)
	}
	if got[0].File != "app/src/main/java/com/example/hascall/App.kt" {
		t.Errorf("suggestions[0].File = %q, want the existing App.kt", got[0].File)
	}
}
