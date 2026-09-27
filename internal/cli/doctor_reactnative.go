package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/ts"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
)

// doctorReactNative runs the spec's F6 checks for React Native: init in
// a useEffect of the component registered with
// AppRegistry.registerComponent, and screen tracking + overlay host for
// every React Navigation route.
//
// The init check is component-level, not useEffect-level: it confirms
// the (single) init CallSite's Enclosing matches the registered root
// component's name, not specifically that it's inside a useEffect
// there — CallSite doesn't retain enough AST context to check the
// useEffect wrapper specifically, the same kind of granularity
// trade-off doctorAndroid already makes at file level.
//
// Screen discovery covers both React Navigation's JSX route form
// (<Stack.Screen name="X" component={Y} />) and its static-API form
// (createXNavigator({ screens: { X: Y } })). A screen registered
// through a render-prop child, or given as an object (for per-screen
// options) instead of a plain component reference, isn't matched (see
// ts.NavigatorScreens and ts.StaticNavigatorScreens) — it doesn't
// appear here at all rather than being guessed at.
func doctorReactNative(app *App, sites []extract.CallSite) ([]DoctorResult, error) {
	inv, err := reactNativeInventory(app)
	if err != nil {
		return nil, err
	}

	results := []DoctorResult{checkReactNativeInit(sites, inv)}
	results = append(results, checkReactNativeScreens(sites, inv)...)
	return results, nil
}

func checkReactNativeInit(sites []extract.CallSite, inv rnInventory) DoctorResult {
	var inits []extract.CallSite
	for _, s := range sites {
		if s.Kind == "init" {
			inits = append(inits, s)
		}
	}
	if len(inits) == 0 {
		return DoctorResult{Check: "init", Screen: "-", File: "-", Status: "fail: no init call found"}
	}
	if len(inits) > 1 {
		return DoctorResult{Check: "init", Screen: "-", File: inits[0].File, Line: inits[0].Line,
			Status: fmt.Sprintf("fail: %d init calls found, expected exactly one", len(inits))}
	}
	site := inits[0]

	if inv.RootComponent == "" {
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line,
			Status: "fail: no AppRegistry.registerComponent call found"}
	}
	if site.Enclosing != inv.RootComponent {
		display := site.Enclosing
		if display == "" {
			display = "(no enclosing component)"
		}
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line,
			Status: fmt.Sprintf("fail: init is in %s, not %s (the component registered with AppRegistry)", display, inv.RootComponent)}
	}
	return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line, Status: "pass"}
}

func checkReactNativeScreens(sites []extract.CallSite, inv rnInventory) []DoctorResult {
	var results []DoctorResult
	for _, route := range inv.Routes {
		var comp *ts.ComponentInfo
		for i := range inv.Components {
			if inv.Components[i].Name == route.ComponentName {
				comp = &inv.Components[i]
				break
			}
		}
		screenName := route.RouteName
		if screenName == "" {
			screenName = route.ComponentName
		}
		if comp == nil {
			results = append(results, DoctorResult{
				Check: "screen", Screen: screenName, File: "-", TrackedAs: "-", Overlay: "-",
				Status: "warn: heuristic: source file for this component was not found",
			})
			continue
		}

		tracked := fileHasKind(sites, comp.File, "screen")
		overlay := fileHasKind(sites, comp.File, "overlay-host")

		trackedAs := "-"
		for _, s := range sites {
			if s.File == comp.File && s.Kind == "screen" {
				if s.Dynamic {
					trackedAs = "<dynamic>"
				} else {
					trackedAs = s.Name
				}
				break
			}
		}
		overlayStatus := "missing"
		if overlay {
			overlayStatus = "root"
		}

		status := "pass"
		switch {
		case !tracked:
			status = "fail: not tracked"
		case !overlay:
			status = "fail: no overlay host"
		}

		results = append(results, DoctorResult{
			Check: "screen", Screen: screenName, File: comp.File, Line: comp.Line,
			TrackedAs: trackedAs, Overlay: overlayStatus, Status: status,
		})
	}
	return results
}

// rnInventory is the groundwork doctor, init and integrate all need for
// React Native, gathered by walking every source file once.
type rnInventory struct {
	Files         []string
	Components    []ts.ComponentInfo
	Routes        []ts.NavRoute
	RootComponent string // AppRegistry.registerComponent's target, "" if not found
}

func reactNativeInventory(app *App) (rnInventory, error) {
	var inv rnInventory

	adapter, _ := project.ByPlatform(project.ReactNative)
	files, err := adapter.Files(app.Root)
	if err != nil {
		return inv, fmt.Errorf("listing files: %w", err)
	}
	inv.Files = files

	pool, err := parse.NewPool(ts.Language(), 0)
	if err != nil {
		return inv, fmt.Errorf("starting parser pool: %w", err)
	}
	defer pool.Close()

	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(app.Root, rel))
		if err != nil {
			return inv, fmt.Errorf("reading %s: %w", rel, err)
		}
		tree, err := pool.Parse(context.Background(), data)
		if err != nil {
			return inv, fmt.Errorf("parsing %s: %w", rel, err)
		}
		inv.Components = append(inv.Components, ts.Components(tree, data, rel)...)
		inv.Routes = append(inv.Routes, ts.NavigatorScreens(tree, data)...)
		inv.Routes = append(inv.Routes, ts.StaticNavigatorScreens(tree, data)...)
		if inv.RootComponent == "" {
			if name, ok := ts.AppRegistryComponent(tree, data); ok {
				inv.RootComponent = name
			}
		}
		tree.Close()
	}
	return inv, nil
}
