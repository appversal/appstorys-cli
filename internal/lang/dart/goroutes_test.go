package dart

import "testing"

const goRouterFixtureSrc = `
final GoRouter router = GoRouter(
  routes: [
    GoRoute(
      path: '/',
      builder: (context, state) => HomeScreen(),
    ),
    GoRoute(
      path: '/settings',
      builder: (context, state) => SettingsScreen(),
      routes: [
        GoRoute(
          path: 'nested',
          builder: (context, state) => NestedScreen(),
        ),
      ],
    ),
  ],
);
`

func TestGoRoutes(t *testing.T) {
	tree := parse(t, goRouterFixtureSrc)
	defer tree.Close()

	routes := GoRoutes(tree, []byte(goRouterFixtureSrc))
	if len(routes) != 3 {
		t.Fatalf("GoRoutes() = %d routes, want 3 (including the nested one): %+v", len(routes), routes)
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
	if byPath["nested"].WidgetName != "NestedScreen" {
		t.Errorf("nested route widget = %q, want NestedScreen (sub-routes should be found too)", byPath["nested"].WidgetName)
	}
}

func TestGoRoutesNone(t *testing.T) {
	src := `void main() { runApp(MyApp()); }`
	tree := parse(t, src)
	defer tree.Close()

	if routes := GoRoutes(tree, []byte(src)); len(routes) != 0 {
		t.Errorf("GoRoutes() = %+v, want none", routes)
	}
}
