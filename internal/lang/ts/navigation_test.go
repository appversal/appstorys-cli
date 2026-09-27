package ts

import "testing"

const navFixtureSrc = `
import { AppRegistry } from 'react-native';
import App from './App';

AppRegistry.registerComponent('MyApp', () => App);

function AppNavigator() {
  return (
    <Stack.Navigator>
      <Stack.Screen name="Home" component={HomeScreen} />
      <Stack.Screen name="Settings" component={SettingsScreen} />
    </Stack.Navigator>
  );
}

const HomeScreen = () => {
  return (
    <View>
      <Text>Home</Text>
    </View>
  );
};

export default function SettingsScreen() {
  return <View><Text>Settings</Text></View>;
}
`

func TestComponents(t *testing.T) {
	tree := parse(t, navFixtureSrc)
	defer tree.Close()

	components := Components(tree, []byte(navFixtureSrc), "App.tsx")
	byName := map[string]ComponentInfo{}
	for _, c := range components {
		byName[c.Name] = c
	}

	for _, name := range []string{"AppNavigator", "HomeScreen", "SettingsScreen"} {
		if _, ok := byName[name]; !ok {
			t.Errorf("component %q not found; got %+v", name, components)
		}
	}

	home := byName["HomeScreen"]
	if home.ReturnJSXStart == 0 || home.ReturnJSXEnd == 0 {
		t.Errorf("HomeScreen ReturnJSX range = [%d,%d], want a resolved range", home.ReturnJSXStart, home.ReturnJSXEnd)
	}
	jsxText := navFixtureSrc[home.ReturnJSXStart:home.ReturnJSXEnd]
	if jsxText[:5] != "<View" {
		t.Errorf("HomeScreen returned JSX text = %q, want it to start with <View", jsxText)
	}

	settings := byName["SettingsScreen"]
	if settings.ReturnJSXStart == 0 || settings.ReturnJSXEnd == 0 {
		t.Errorf("SettingsScreen ReturnJSX range = [%d,%d], want a resolved range", settings.ReturnJSXStart, settings.ReturnJSXEnd)
	}

	// AppNavigator's return is a nested Stack.Navigator/.Screen tree,
	// not a plain component's own markup — still a "simple shape" by
	// our rule (a single top-level JSX expression), which is fine:
	// nothing will ever propose wrapping AppNavigator itself since it's
	// never named as a NavRoute's ComponentName.
	nav := byName["AppNavigator"]
	if nav.ReturnJSXStart == 0 {
		t.Errorf("AppNavigator ReturnJSX range unresolved, want resolved (single parenthesized JSX return)")
	}
}

func TestAppRegistryComponent(t *testing.T) {
	tree := parse(t, navFixtureSrc)
	defer tree.Close()

	name, found := AppRegistryComponent(tree, []byte(navFixtureSrc))
	if !found || name != "App" {
		t.Errorf("AppRegistryComponent() = (%q, %v), want (\"App\", true)", name, found)
	}
}

func TestAppRegistryComponentBareIdentifier(t *testing.T) {
	src := `AppRegistry.registerComponent('MyApp', App);`
	tree := parse(t, src)
	defer tree.Close()

	name, found := AppRegistryComponent(tree, []byte(src))
	if !found || name != "App" {
		t.Errorf("AppRegistryComponent() = (%q, %v), want (\"App\", true)", name, found)
	}
}

func TestAppRegistryComponentNotFound(t *testing.T) {
	src := `function App() { return null; }`
	tree := parse(t, src)
	defer tree.Close()

	if _, found := AppRegistryComponent(tree, []byte(src)); found {
		t.Error("AppRegistryComponent() found = true, want false")
	}
}

func TestNavigatorScreens(t *testing.T) {
	tree := parse(t, navFixtureSrc)
	defer tree.Close()

	routes := NavigatorScreens(tree, []byte(navFixtureSrc))
	if len(routes) != 2 {
		t.Fatalf("NavigatorScreens() = %d routes, want 2: %+v", len(routes), routes)
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
