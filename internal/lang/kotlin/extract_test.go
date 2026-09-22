package kotlin

import (
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

func parse(t *testing.T, src string) *sitter.Tree {
	t.Helper()
	parser := sitter.NewParser()
	t.Cleanup(parser.Close)
	if err := parser.SetLanguage(Language()); err != nil {
		t.Fatal(err)
	}
	return parser.Parse([]byte(src), nil)
}

func TestExtractAndMatch(t *testing.T) {
	src := `
class LoginViewModel {
    fun submit() {
        AppStorys.trackEvents(event = "Login", metadata = mapOf("method" to "email"))
        AppStorys.getInstance().getScreenCampaigns("Home Screen", listOf("widget_one"))
    }
}

@Composable
fun MainActivity() {
    overlayElements()
    Widget(position = "widget_one")
    Modifier.appstorys("tooltip_one")
}
`
	tree := parse(t, src)
	defer tree.Close()

	calls := Extract(tree, []byte(src))
	if len(calls) == 0 {
		t.Fatal("Extract() returned no calls")
	}

	m, err := symbols.LoadPlatform("android")
	if err != nil {
		t.Fatalf("LoadPlatform(android) error = %v", err)
	}

	sites := extract.Match(calls, "Login.kt", []byte(src), project.Android, m, LiteralReader{}, extract.Options{})

	byKind := map[string][]extract.CallSite{}
	for _, s := range sites {
		byKind[s.Kind] = append(byKind[s.Kind], s)
	}

	events := byKind["event"]
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1: %+v", len(events), events)
	}
	if events[0].Name != "Login" {
		t.Errorf("event Name = %q, want %q", events[0].Name, "Login")
	}
	if events[0].Enclosing != "LoginViewModel.submit" {
		t.Errorf("event Enclosing = %q, want %q", events[0].Enclosing, "LoginViewModel.submit")
	}
	if pt := events[0].Properties["method"]; pt != extract.PropString {
		t.Errorf("event Properties[method] = %q, want %q", pt, extract.PropString)
	}

	screens := byKind["screen"]
	if len(screens) != 1 {
		t.Fatalf("screens = %d, want 1: %+v", len(screens), screens)
	}
	if screens[0].Name != "Home Screen" {
		t.Errorf("screen Name = %q, want %q", screens[0].Name, "Home Screen")
	}
	if screens[0].Position != "widget_one" {
		t.Errorf("screen Position = %q, want %q", screens[0].Position, "widget_one")
	}

	if len(byKind["overlay-host"]) != 1 {
		t.Errorf("overlay-host = %d, want 1", len(byKind["overlay-host"]))
	}

	placements := byKind["placement"]
	if len(placements) != 1 || placements[0].Position != "widget_one" {
		t.Errorf("placements = %+v, want one Widget at widget_one", placements)
	}

	tags := byKind["tag"]
	if len(tags) != 1 || tags[0].Name != "tooltip_one" {
		t.Errorf("tags = %+v, want one tooltip_one", tags)
	}
}

func TestExtractDynamicName(t *testing.T) {
	src := `
fun helper(name: String) {
    AppStorys.trackEvents(event = name)
}
`
	tree := parse(t, src)
	defer tree.Close()

	m, err := symbols.LoadPlatform("android")
	if err != nil {
		t.Fatalf("LoadPlatform(android) error = %v", err)
	}

	sites := extract.Match(Extract(tree, []byte(src)), "helper.kt", []byte(src), project.Android, m, LiteralReader{}, extract.Options{})
	if len(sites) != 1 {
		t.Fatalf("sites = %d, want 1", len(sites))
	}
	if !sites[0].Dynamic {
		t.Errorf("Dynamic = false, want true for a non-literal event name")
	}
}
