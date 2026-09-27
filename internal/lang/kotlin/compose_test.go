package kotlin

import "testing"

func TestComposeNavHosts(t *testing.T) {
	src := `
@Composable
fun AppNavHost() {
    val navController = rememberNavController()
    NavHost(navController = navController, startDestination = "home") {
        composable("home") {
            HomeScreen()
        }
        composable("settings") {
            SettingsScreen()
        }
    }
}
`
	tree := parse(t, src)
	defer tree.Close()

	hosts := ComposeNavHosts(tree, []byte(src))
	if len(hosts) != 1 {
		t.Fatalf("ComposeNavHosts() = %d hosts, want 1: %+v", len(hosts), hosts)
	}
	h := hosts[0]
	if h.EnclosingFunc != "AppNavHost" {
		t.Errorf("EnclosingFunc = %q, want %q", h.EnclosingFunc, "AppNavHost")
	}
	if src[h.EnclosingBodyInsertAt-1] != '{' {
		t.Errorf("EnclosingBodyInsertAt = %d points at %q, want right after function's '{'",
			h.EnclosingBodyInsertAt, string(src[h.EnclosingBodyInsertAt-1]))
	}

	if len(h.Destinations) != 2 {
		t.Fatalf("Destinations = %d, want 2: %+v", len(h.Destinations), h.Destinations)
	}
	if h.Destinations[0].Route != "home" || h.Destinations[0].Dynamic {
		t.Errorf("Destinations[0] = %+v, want route \"home\"", h.Destinations[0])
	}
	if h.Destinations[1].Route != "settings" {
		t.Errorf("Destinations[1] = %+v, want route \"settings\"", h.Destinations[1])
	}
	if src[h.Destinations[0].BodyInsertAt-1] != '{' {
		t.Errorf("Destinations[0].BodyInsertAt = %d points at %q, want right after its lambda '{'",
			h.Destinations[0].BodyInsertAt, string(src[h.Destinations[0].BodyInsertAt-1]))
	}
	if h.Destinations[0].StartLine >= h.Destinations[1].StartLine {
		t.Errorf("Destinations out of order: %+v", h.Destinations)
	}
}

func TestComposeNavHostsDynamicRoute(t *testing.T) {
	src := `
fun Nav() {
    NavHost(navController = nav, startDestination = start) {
        composable(routeVar) {
            Screen()
        }
    }
}
`
	tree := parse(t, src)
	defer tree.Close()

	hosts := ComposeNavHosts(tree, []byte(src))
	if len(hosts) != 1 || len(hosts[0].Destinations) != 1 {
		t.Fatalf("unexpected hosts: %+v", hosts)
	}
	if !hosts[0].Destinations[0].Dynamic {
		t.Errorf("Destinations[0].Dynamic = false, want true for a non-literal route")
	}
}

func TestComposeNavHostsNone(t *testing.T) {
	src := `fun NoNav() { Text("hi") }`
	tree := parse(t, src)
	defer tree.Close()

	if hosts := ComposeNavHosts(tree, []byte(src)); len(hosts) != 0 {
		t.Errorf("ComposeNavHosts() = %+v, want none", hosts)
	}
}
