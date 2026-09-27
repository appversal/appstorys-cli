package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/ts"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// planInitReactNative proposes the AppStorys.initialize(...) call in the
// root component's mount-only useEffect (creating one if it doesn't
// have one), per the spec's "Root component useEffect" placement rule.
//
// Only the init call itself is proposed. Dependency wiring (the npm
// package and its 14 peer dependencies via the detected package
// manager, the Babel plugin, GestureHandlerRootView + SafeAreaProvider
// at root, appstorys.d.ts in tsconfig.json) isn't generated — each is
// its own mechanism (package.json/lockfile editing, Babel config,
// wrapping existing root JSX, tsconfig editing) genuinely distinct from
// anything built so far, left as documented manual steps rather than
// rushed.
func planInitReactNative(app *App, sites []extract.CallSite) ([]suggest.Suggestion, error) {
	for _, s := range sites {
		if s.Kind == "init" {
			return nil, nil // won't move an existing (possibly misplaced) call, same policy as Android
		}
	}

	inv, err := reactNativeInventory(app)
	if err != nil {
		return nil, err
	}
	if inv.RootComponent == "" {
		return nil, &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf(
			"no AppRegistry.registerComponent call found; can't determine where to add init")}
	}

	var compInfo *ts.ComponentInfo
	for i := range inv.Components {
		if inv.Components[i].Name == inv.RootComponent {
			compInfo = &inv.Components[i]
			break
		}
	}
	if compInfo == nil {
		return nil, &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf(
			"could not find the source of component %q registered with AppRegistry", inv.RootComponent)}
	}

	data, err := os.ReadFile(filepath.Join(app.Root, compInfo.File))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", compInfo.File, err)
	}
	pool, err := parse.NewPool(ts.Language(), 1)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	tree, err := pool.Parse(context.Background(), data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", compInfo.File, err)
	}
	defer tree.Close()

	at, hasUseEffect, found := ts.LocateComponentInit(tree, data, inv.RootComponent)
	if !found {
		return nil, &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf(
			"%s's body isn't a plain function/arrow-function block; add the init call manually", inv.RootComponent)}
	}

	tokenExpr := app.Config.Init.TokenExpr
	requires := []string{"wire the npm dependency + Babel plugin + root wrappers (not yet automated)"}
	if tokenExpr == "" {
		tokenExpr = "APPSTORYS_API_TOKEN"
		requires = append(requires, "define APPSTORYS_API_TOKEN (e.g. from react-native-config or your own env setup)")
	}
	call := fmt.Sprintf("AppStorys.initialize(%s);", tokenExpr)

	var snippet, reason string
	if hasUseEffect {
		snippet = "\n    " + call
		reason = fmt.Sprintf("Add the AppStorys init call to %s's existing mount-only useEffect", inv.RootComponent)
	} else {
		snippet = fmt.Sprintf("\n  useEffect(() => {\n    %s\n  }, []);\n", call)
		reason = fmt.Sprintf("%s has no mount-only useEffect; adding one with the AppStorys init call", inv.RootComponent)
		requires = append(requires, "import useEffect from react if not already imported")
	}

	return []suggest.Suggestion{{
		ID:         suggest.StableID("init", compInfo.File, "useEffect-init-call"),
		Kind:       "init",
		File:       compInfo.File,
		InsertAt:   at,
		Snippet:    snippet,
		Reason:     reason,
		Confidence: 0.85,
		Requires:   requires,
	}}, nil
}
