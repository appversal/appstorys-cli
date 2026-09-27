package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/dart"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
)

// parseDartFile reads and parses root/file, returning the tree, its
// source bytes, and a cleanup function that closes both the tree and
// the single-parser pool backing it.
func parseDartFile(root, file string) (tree *sitter.Tree, data []byte, closeFn func(), err error) {
	data, err = os.ReadFile(filepath.Join(root, file))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("reading %s: %w", file, err)
	}
	pool, err := parse.NewPool(dart.Language(), 1)
	if err != nil {
		return nil, nil, nil, err
	}
	tree, err = pool.Parse(context.Background(), data)
	if err != nil {
		pool.Close()
		return nil, nil, nil, fmt.Errorf("parsing %s: %w", file, err)
	}
	return tree, data, func() { tree.Close(); pool.Close() }, nil
}

// doctorFlutter runs the spec's F6 checks for Flutter: init in main()
// or the root widget's State.initState, and screen tracking + overlay
// host for every MaterialApp(routes: {...}) entry and every GoRoute
// (go_router), including nested ones.
//
// Screen discovery only covers those two mechanisms — the spec's other
// two (onGenerateRoute, Navigator.push) aren't matched (see
// dart.MaterialAppRoutes and dart.GoRoutes), so screens registered
// those ways don't appear here at all.
//
// A route widget's file for tracking/overlay correlation is its State
// class's file if it's a StatefulWidget (matching the spec's
// "State.initState" requirement), or the widget's own file if it's
// stateless — file-level correlation, the same heuristic granularity
// doctorAndroid and doctorReactNative already use.
func doctorFlutter(app *App, sites []extract.CallSite) ([]DoctorResult, error) {
	inv, err := flutterInventoryFor(app)
	if err != nil {
		return nil, err
	}

	results := []DoctorResult{checkFlutterInit(sites, inv)}
	results = append(results, checkFlutterScreens(sites, inv)...)
	return results, nil
}

func checkFlutterInit(sites []extract.CallSite, inv flutterInventory) DoctorResult {
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

	if site.Enclosing == "main" {
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line, Status: "pass"}
	}

	if inv.RootWidget == "" {
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line,
			Status: "fail: no runApp(...) call found to identify the root widget"}
	}

	var rootState string
	for _, c := range inv.Classes {
		if c.Supertype == "State" && c.StateOf == inv.RootWidget {
			rootState = c.Name
			break
		}
	}
	if rootState != "" && site.Enclosing == rootState+".initState" {
		return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line, Status: "pass"}
	}

	display := site.Enclosing
	if display == "" {
		display = "(no enclosing function)"
	}
	return DoctorResult{Check: "init", Screen: "-", File: site.File, Line: site.Line,
		Status: fmt.Sprintf("fail: init is in %s, not main() or %s's initState", display, inv.RootWidget)}
}

func checkFlutterScreens(sites []extract.CallSite, inv flutterInventory) []DoctorResult {
	var results []DoctorResult
	for _, route := range inv.Routes {
		if route.WidgetName == "" {
			continue // builder shape we couldn't resolve; not silently guessed
		}
		file, line, ok := flutterScreenFile(inv.Classes, route.WidgetName)
		if !ok {
			results = append(results, DoctorResult{
				Check: "screen", Screen: route.WidgetName, File: "-", TrackedAs: "-", Overlay: "-",
				Status: "warn: heuristic: source file for this widget was not found",
			})
			continue
		}

		tracked := fileHasKind(sites, file, "screen")
		overlay := fileHasKind(sites, file, "overlay-host")

		trackedAs := "-"
		for _, s := range sites {
			if s.File == file && s.Kind == "screen" {
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
			Check: "screen", Screen: route.WidgetName, File: file, Line: line,
			TrackedAs: trackedAs, Overlay: overlayStatus, Status: status,
		})
	}
	return results
}

// flutterScreenFile resolves a widget name to the file doctor/integrate
// should treat as "this screen's source": its State class's file for a
// StatefulWidget, or the widget's own file for a StatelessWidget.
func flutterScreenFile(classes []dart.ClassInfo, widgetName string) (file string, line int, ok bool) {
	var widgetClass *dart.ClassInfo
	for i := range classes {
		if classes[i].Name == widgetName {
			widgetClass = &classes[i]
			break
		}
	}
	if widgetClass == nil {
		return "", 0, false
	}
	if widgetClass.Supertype == "StatefulWidget" {
		for _, c := range classes {
			if c.Supertype == "State" && c.StateOf == widgetName {
				return c.File, c.Line, true
			}
		}
		return "", 0, false
	}
	return widgetClass.File, widgetClass.Line, true
}

// flutterInventory is the groundwork doctor, init and integrate all
// need for Flutter, gathered by walking every source file once.
type flutterInventory struct {
	Files      []string
	Classes    []dart.ClassInfo
	Routes     []dart.Route
	RootWidget string
	MainFile   string
}

func flutterInventoryFor(app *App) (flutterInventory, error) {
	var inv flutterInventory

	adapter, _ := project.ByPlatform(project.Flutter)
	files, err := adapter.Files(app.Root)
	if err != nil {
		return inv, fmt.Errorf("listing files: %w", err)
	}
	inv.Files = files

	pool, err := parse.NewPool(dart.Language(), 0)
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
		inv.Classes = append(inv.Classes, dart.Classes(tree, data, rel)...)
		inv.Routes = append(inv.Routes, dart.MaterialAppRoutes(tree, data)...)
		inv.Routes = append(inv.Routes, dart.GoRoutes(tree, data)...)
		if inv.RootWidget == "" {
			if name, ok := dart.RunAppWidget(tree, data); ok {
				inv.RootWidget = name
			}
		}
		if inv.MainFile == "" {
			if _, ok := dart.LocateTopLevelFunction(tree, data, "main"); ok {
				inv.MainFile = rel
			}
		}
		tree.Close()
	}
	return inv, nil
}
