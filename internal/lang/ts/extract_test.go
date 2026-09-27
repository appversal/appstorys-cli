package ts

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

const fixtureSrc = `
import AppStorys from '@appstorys/appstorys-react-native';

function App() {
  useEffect(() => {
    AppStorys.initialize('token123');
  }, []);

  function submit() {
    AppStorys.trackEvent('Login', undefined, { method: 'email' });
  }

  return (
    <AppStorys.Screen name="Home Screen" options={{ positionList: ['widget_one'] }}>
      <View appstorys="tooltip_one">
        <AppStorys.Widgets position="widget_one" />
        <AppStorys.Stories />
      </View>
    </AppStorys.Screen>
  );
}
`

func TestExtractAndMatch(t *testing.T) {
	tree := parse(t, fixtureSrc)
	defer tree.Close()
	src := []byte(fixtureSrc)

	calls := Extract(tree, src)
	if len(calls) == 0 {
		t.Fatal("Extract() returned no calls")
	}

	m, err := symbols.LoadPlatform("react-native")
	if err != nil {
		t.Fatalf("LoadPlatform(react-native) error = %v", err)
	}

	sites := extract.Match(calls, "App.tsx", src, project.ReactNative, m, LiteralReader{}, extract.Options{})
	sites = append(sites, ScanTags(tree, src, "App.tsx")...)

	byKind := map[string][]extract.CallSite{}
	for _, s := range sites {
		byKind[s.Kind] = append(byKind[s.Kind], s)
	}

	if len(byKind["init"]) != 1 {
		t.Errorf("init sites = %d, want 1: %+v", len(byKind["init"]), byKind["init"])
	}

	events := byKind["event"]
	if len(events) != 1 {
		t.Fatalf("event sites = %d, want 1: %+v", len(events), events)
	}
	if events[0].Name != "Login" {
		t.Errorf("event Name = %q, want %q", events[0].Name, "Login")
	}
	if events[0].Enclosing != "submit" {
		t.Errorf("event Enclosing = %q, want %q", events[0].Enclosing, "submit")
	}
	if pt := events[0].Properties["method"]; pt != extract.PropString {
		t.Errorf("event Properties[method] = %q, want %q", pt, extract.PropString)
	}

	screens := byKind["screen"]
	if len(screens) != 1 || screens[0].Name != "Home Screen" {
		t.Fatalf("screen sites = %+v, want one \"Home Screen\"", screens)
	}

	overlay := byKind["overlay-host"]
	if len(overlay) != 1 {
		t.Fatalf("overlay-host sites = %d, want 1: %+v", len(overlay), overlay)
	}
	// The overlay-host site should be the SAME <AppStorys.Screen>
	// element as the screen site (same line), not a different node.
	if overlay[0].Line != screens[0].Line {
		t.Errorf("overlay-host Line = %d, screen Line = %d, want equal (same <AppStorys.Screen> element)",
			overlay[0].Line, screens[0].Line)
	}

	placements := byKind["placement"]
	if len(placements) != 2 {
		t.Fatalf("placement sites = %d, want 2 (Widgets + Stories): %+v", len(placements), placements)
	}
	var sawWidget, sawStories bool
	for _, p := range placements {
		switch p.Via {
		case "AppStorys.Widgets":
			sawWidget = true
			if p.Position != "widget_one" {
				t.Errorf("Widgets Position = %q, want widget_one", p.Position)
			}
		case "AppStorys.Stories":
			sawStories = true
		}
	}
	if !sawWidget || !sawStories {
		t.Errorf("placements = %+v, want both Widgets and Stories", placements)
	}

	tags := byKind["tag"]
	if len(tags) != 1 || tags[0].Name != "tooltip_one" {
		t.Fatalf("tag sites = %+v, want one tooltip_one", tags)
	}
}

func TestExtractDynamicEventName(t *testing.T) {
	src := `
function helper(name) {
  AppStorys.trackEvent(name);
}
`
	tree := parse(t, src)
	defer tree.Close()

	m, err := symbols.LoadPlatform("react-native")
	if err != nil {
		t.Fatalf("LoadPlatform(react-native) error = %v", err)
	}

	sites := extract.Match(Extract(tree, []byte(src)), "helper.ts", []byte(src), project.ReactNative, m, LiteralReader{}, extract.Options{})
	if len(sites) != 1 {
		t.Fatalf("sites = %d, want 1", len(sites))
	}
	if !sites[0].Dynamic {
		t.Errorf("Dynamic = false, want true for a non-literal event name")
	}
}
