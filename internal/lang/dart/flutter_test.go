package dart

import "testing"

const flutterFixtureSrc = `
void main() {
  WidgetsFlutterBinding.ensureInitialized();
  AppStorys.initialize('token123');
  runApp(MyApp());
}

class MyApp extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      routes: {
        '/': (context) => HomeScreen(),
        '/settings': (context) => SettingsScreen(),
      },
    );
  }
}

class HomeScreen extends StatefulWidget {
  @override
  State<HomeScreen> createState() => _HomeScreenState();
}

class _HomeScreenState extends State<HomeScreen> {
  @override
  void initState() {
    super.initState();
    AppStorys.trackScreen('Home Screen', context);
  }

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        Text('Home'),
        ...AppStorys.overlayElements(),
      ],
    );
  }
}
`

func TestClasses(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	classes := Classes(tree, []byte(flutterFixtureSrc), "main.dart")
	byName := map[string]ClassInfo{}
	for _, c := range classes {
		byName[c.Name] = c
	}

	if byName["MyApp"].Supertype != "StatelessWidget" {
		t.Errorf("MyApp Supertype = %q, want StatelessWidget", byName["MyApp"].Supertype)
	}
	if byName["HomeScreen"].Supertype != "StatefulWidget" {
		t.Errorf("HomeScreen Supertype = %q, want StatefulWidget", byName["HomeScreen"].Supertype)
	}
	state := byName["_HomeScreenState"]
	if state.Supertype != "State" || state.StateOf != "HomeScreen" {
		t.Errorf("_HomeScreenState = %+v, want Supertype=State StateOf=HomeScreen", state)
	}
}

func TestLocateClassMethodExisting(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	at, found := LocateClassMethod(tree, []byte(flutterFixtureSrc), "_HomeScreenState", "initState")
	if !found {
		t.Fatal("LocateClassMethod() found = false, want true")
	}
	if flutterFixtureSrc[at-1] != '{' {
		t.Errorf("at = %d points at %q, want right after initState's '{'", at, string(flutterFixtureSrc[at-1]))
	}
}

func TestLocateClassMethodArrowBody(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	// createState() => _HomeScreenState(); has no block body.
	if _, found := LocateClassMethod(tree, []byte(flutterFixtureSrc), "HomeScreen", "createState"); found {
		t.Error("LocateClassMethod() found = true, want false for an arrow-body method")
	}
}

func TestLocateClassMethodMissing(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	if _, found := LocateClassMethod(tree, []byte(flutterFixtureSrc), "_HomeScreenState", "onResume"); found {
		t.Error("LocateClassMethod() found = true, want false for a missing method")
	}
}

func TestLocateClassBody(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	at, found := LocateClassBody(tree, []byte(flutterFixtureSrc), "_HomeScreenState")
	if !found {
		t.Fatal("LocateClassBody() found = false, want true")
	}
	if flutterFixtureSrc[at-1] != '{' {
		t.Errorf("at = %d points at %q, want right after class '{'", at, string(flutterFixtureSrc[at-1]))
	}
}

func TestLocateTopLevelFunction(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	at, found := LocateTopLevelFunction(tree, []byte(flutterFixtureSrc), "main")
	if !found {
		t.Fatal("LocateTopLevelFunction() found = false, want true")
	}
	if flutterFixtureSrc[at-1] != '{' {
		t.Errorf("at = %d points at %q, want right after main's '{'", at, string(flutterFixtureSrc[at-1]))
	}

	if _, found := LocateTopLevelFunction(tree, []byte(flutterFixtureSrc), "notThere"); found {
		t.Error("LocateTopLevelFunction() found = true, want false for a missing function")
	}
}

func TestRunAppWidget(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	name, found := RunAppWidget(tree, []byte(flutterFixtureSrc))
	if !found || name != "MyApp" {
		t.Errorf("RunAppWidget() = (%q, %v), want (\"MyApp\", true)", name, found)
	}
}

func TestStackChildrenInsertPoint(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	at, found := StackChildrenInsertPoint(tree, []byte(flutterFixtureSrc))
	if !found {
		t.Fatal("StackChildrenInsertPoint() found = false, want true")
	}
	if flutterFixtureSrc[at] != ']' {
		t.Errorf("at = %d points at %q, want ']'", at, string(flutterFixtureSrc[at]))
	}
}

func TestStackChildrenInsertPointMissing(t *testing.T) {
	src := `Widget build(BuildContext context) { return Text('hi'); }`
	tree := parse(t, src)
	defer tree.Close()

	if _, found := StackChildrenInsertPoint(tree, []byte(src)); found {
		t.Error("StackChildrenInsertPoint() found = true, want false when there's no Stack")
	}
}

func TestMaterialAppRoutes(t *testing.T) {
	tree := parse(t, flutterFixtureSrc)
	defer tree.Close()

	routes := MaterialAppRoutes(tree, []byte(flutterFixtureSrc))
	if len(routes) != 2 {
		t.Fatalf("MaterialAppRoutes() = %d routes, want 2: %+v", len(routes), routes)
	}
	byPath := map[string]Route{}
	for _, r := range routes {
		byPath[r.Path] = r
	}
	if byPath["/"].WidgetName != "HomeScreen" {
		t.Errorf("route / widget = %q, want HomeScreen", byPath["/"].WidgetName)
	}
	if byPath["/settings"].WidgetName != "SettingsScreen" {
		t.Errorf("route /settings widget = %q, want SettingsScreen", byPath["/settings"].WidgetName)
	}
}
