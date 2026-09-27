package cli

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/catalog"
	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/rules"
)

func TestScanProjectReactNative(t *testing.T) {
	app := testApp(t, "../../testdata/fixtures/reactnative/sample-app")

	sites, platforms, err := scanProject(app, nil)
	if err != nil {
		t.Fatalf("scanProject() error = %v", err)
	}
	if len(platforms) != 1 || platforms[0] != "react-native" {
		t.Fatalf("platforms = %v, want [react-native]", platforms)
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
	// <AppStorys.Screen> serves both concepts from the same element.
	if len(byKind["overlay-host"]) != 1 || byKind["overlay-host"][0].Line != byKind["screen"][0].Line {
		t.Errorf("overlay-host sites = %+v, want one on the same line as the screen site", byKind["overlay-host"])
	}
	if len(byKind["event"]) != 2 { // Login + clicked
		t.Errorf("event sites = %d, want 2: %+v", len(byKind["event"]), byKind["event"])
	}
	if len(byKind["placement"]) != 2 { // Widgets + Stories
		t.Errorf("placement sites = %d, want 2: %+v", len(byKind["placement"]), byKind["placement"])
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
	if !severityAtOrAbove(findings, rules.SeverityError) {
		t.Errorf("expected at least one error-level finding (reserved-event-name)")
	}
}
