package ts

import "testing"

func TestStaticNavigatorScreens(t *testing.T) {
	src := `
const RootStack = createNativeStackNavigator({
  screens: {
    Home: HomeScreen,
    Settings: SettingsScreen,
  },
});
`
	tree := parse(t, src)
	defer tree.Close()

	routes := StaticNavigatorScreens(tree, []byte(src))
	if len(routes) != 2 {
		t.Fatalf("StaticNavigatorScreens() = %d routes, want 2: %+v", len(routes), routes)
	}
	byRoute := map[string]NavRoute{}
	for _, r := range routes {
		byRoute[r.RouteName] = r
	}
	if byRoute["Home"].ComponentName != "HomeScreen" {
		t.Errorf("Home route component = %q, want HomeScreen", byRoute["Home"].ComponentName)
	}
	if byRoute["Settings"].ComponentName != "SettingsScreen" {
		t.Errorf("Settings route component = %q, want SettingsScreen", byRoute["Settings"].ComponentName)
	}
}

func TestStaticNavigatorScreensStringKeys(t *testing.T) {
	src := `
const RootStack = createBottomTabNavigator({
  screens: {
    'Home': HomeScreen,
  },
});
`
	tree := parse(t, src)
	defer tree.Close()

	routes := StaticNavigatorScreens(tree, []byte(src))
	if len(routes) != 1 || routes[0].RouteName != "Home" || routes[0].ComponentName != "HomeScreen" {
		t.Errorf("routes = %+v, want one Home -> HomeScreen", routes)
	}
}

func TestStaticNavigatorScreensSkipsNonIdentifierValues(t *testing.T) {
	src := `
const RootStack = createNativeStackNavigator({
  screens: {
    Home: HomeScreen,
    Settings: { screen: SettingsScreen, options: { title: 'Settings' } },
  },
});
`
	tree := parse(t, src)
	defer tree.Close()

	routes := StaticNavigatorScreens(tree, []byte(src))
	if len(routes) != 1 || routes[0].RouteName != "Home" {
		t.Errorf("routes = %+v, want only Home (Settings' object-shaped config isn't matched)", routes)
	}
}

func TestStaticNavigatorScreensIgnoresNonNavigatorCalls(t *testing.T) {
	src := `
const config = buildSomething({
  screens: {
    Home: HomeScreen,
  },
});
`
	tree := parse(t, src)
	defer tree.Close()

	if routes := StaticNavigatorScreens(tree, []byte(src)); len(routes) != 0 {
		t.Errorf("routes = %+v, want none (buildSomething isn't a create*Navigator call)", routes)
	}
}
