package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/catalog"
	"github.com/appversal/appstorys-cli/internal/config"
	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/rules"
)

func testApp(t *testing.T, root string) *App {
	t.Helper()
	cfg := config.Default()
	return &App{Config: &cfg, Root: root}
}

func TestScanProjectAndroid(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/android/sample-app")

	sites, platforms, err := scanProject(app, nil)
	if err != nil {
		t.Fatalf("scanProject() error = %v", err)
	}
	if len(platforms) != 1 || platforms[0] != "android" {
		t.Fatalf("platforms = %v, want [android]", platforms)
	}

	byKind := map[string][]extract.CallSite{}
	for _, s := range sites {
		byKind[s.Kind] = append(byKind[s.Kind], s)
	}
	if len(byKind["init"]) != 1 {
		t.Errorf("init sites = %d, want 1", len(byKind["init"]))
	}
	if len(byKind["screen"]) != 1 || byKind["screen"][0].Name != "Home Screen" {
		t.Errorf("screen sites = %+v", byKind["screen"])
	}
	if len(byKind["event"]) != 2 { // Login + clicked
		t.Errorf("event sites = %d, want 2: %+v", len(byKind["event"]), byKind["event"])
	}
	if len(byKind["overlay-host"]) != 1 {
		t.Errorf("overlay-host sites = %d, want 1", len(byKind["overlay-host"]))
	}
	if len(byKind["placement"]) != 1 {
		t.Errorf("placement sites = %d, want 1", len(byKind["placement"]))
	}
	if len(byKind["tag"]) != 1 || byKind["tag"][0].Name != "tooltip_one" {
		t.Errorf("tag sites = %+v", byKind["tag"])
	}

	c := catalog.Build(sites)
	if len(c.Events) != 2 {
		t.Errorf("catalog events = %d, want 2", len(c.Events))
	}
	if len(c.Screens) != 1 {
		t.Errorf("catalog screens = %d, want 1", len(c.Screens))
	}

	findings, err := lintProject(app, nil)
	if err != nil {
		t.Fatalf("lintProject() error = %v", err)
	}
	var foundReserved bool
	for _, f := range findings {
		if f.RuleID == "reserved-event-name" {
			foundReserved = true
		}
	}
	if !foundReserved {
		t.Errorf("expected a reserved-event-name finding for the \"clicked\" event, findings = %+v", findings)
	}
	if severityAtOrAbove(findings, rules.SeverityError) != true {
		t.Errorf("expected at least one error-level finding (reserved-event-name)")
	}
}

func TestScanProjectFlutter(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/flutter/sample-app")

	sites, platforms, err := scanProject(app, nil)
	if err != nil {
		t.Fatalf("scanProject() error = %v", err)
	}
	if len(platforms) != 1 || platforms[0] != "flutter" {
		t.Fatalf("platforms = %v, want [flutter]", platforms)
	}

	byKind := map[string][]extract.CallSite{}
	for _, s := range sites {
		byKind[s.Kind] = append(byKind[s.Kind], s)
	}
	for _, kind := range []string{"init", "screen", "event", "overlay-host", "placement", "tag"} {
		if len(byKind[kind]) == 0 {
			t.Errorf("no %s call sites found in flutter fixture", kind)
		}
	}

	findings, err := lintProject(app, nil)
	if err != nil {
		t.Fatalf("lintProject() error = %v", err)
	}
	if severityAtOrAbove(findings, rules.SeverityError) {
		t.Errorf("unexpected error-level finding in a clean flutter fixture: %+v", findings)
	}
}
