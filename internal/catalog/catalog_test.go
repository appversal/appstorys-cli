package catalog

import (
	"testing"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/project"
)

func TestBuild(t *testing.T) {
	sites := []extract.CallSite{
		{Kind: "event", Name: "Login", Platform: project.Android, Properties: map[string]extract.PropType{"method": extract.PropString}},
		{Kind: "event", Name: "Login", Platform: project.Flutter},
		{Kind: "event", Dynamic: true, Platform: project.Android},

		{Kind: "screen", Name: "Home Screen", Platform: project.Android, Position: "widget_one,widget_two"},

		{Kind: "placement", Position: "widget_one", Platform: project.Android},
		{Kind: "placement", Position: "widget_three", Platform: project.Flutter},
	}

	c := Build(sites)

	if len(c.Events) != 1 {
		t.Fatalf("Events = %d, want 1", len(c.Events))
	}
	login := c.Events[0]
	if login.CallSites != 2 {
		t.Errorf("Login CallSites = %d, want 2", login.CallSites)
	}
	if len(login.Platforms) != 2 {
		t.Errorf("Login Platforms = %v, want both android and flutter", login.Platforms)
	}
	if login.Properties["method"] != extract.PropString {
		t.Errorf("Login Properties[method] = %v, want string", login.Properties["method"])
	}

	if len(c.Dynamic) != 1 {
		t.Errorf("Dynamic = %d, want 1", len(c.Dynamic))
	}

	if len(c.Screens) != 1 || c.Screens[0].Name != "Home Screen" {
		t.Errorf("Screens = %+v, want one Home Screen", c.Screens)
	}

	wantPositions := map[string]bool{"widget_one": true, "widget_two": true, "widget_three": true}
	if len(c.Positions) != len(wantPositions) {
		t.Fatalf("Positions = %d, want %d: %+v", len(c.Positions), len(wantPositions), c.Positions)
	}
	for _, p := range c.Positions {
		if !wantPositions[p.Name] {
			t.Errorf("unexpected position %q", p.Name)
		}
	}
}
