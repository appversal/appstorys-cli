package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/dart"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// planInitFlutter proposes the AppStorys.initialize(...) call as the
// first statement of main(), per the spec's proposed placement ("main()
// after WidgetsFlutterBinding.ensureInitialized(), before runApp, not
// awaited"). It doesn't verify the call lands specifically after an
// existing WidgetsFlutterBinding.ensureInitialized() call — it's
// inserted as main()'s first statement regardless, the same
// first-statement simplification planInitAndroid already makes for
// Application.onCreate.
//
// Only the init call itself is proposed. Dependency wiring
// (appstorys_sdk_3_0 in pubspec.yaml, Android core library desugaring
// for flutter_local_notifications) isn't generated — pubspec.yaml and
// Gradle files are different editing mechanisms from anything built so
// far, left as documented manual steps.
func planInitFlutter(app *App, sites []extract.CallSite) ([]suggest.Suggestion, error) {
	for _, s := range sites {
		if s.Kind == "init" {
			return nil, nil // won't move an existing (possibly misplaced) call, same policy as Android/RN
		}
	}

	inv, err := flutterInventoryFor(app)
	if err != nil {
		return nil, err
	}
	if inv.MainFile == "" {
		return nil, &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf("no main() function found")}
	}

	data, err := os.ReadFile(filepath.Join(app.Root, inv.MainFile))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", inv.MainFile, err)
	}
	pool, err := parse.NewPool(dart.Language(), 1)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	tree, err := pool.Parse(context.Background(), data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", inv.MainFile, err)
	}
	defer tree.Close()

	at, found := dart.LocateTopLevelFunction(tree, data, "main")
	if !found {
		return nil, fmt.Errorf("could not re-locate main() in %s", inv.MainFile)
	}

	tokenExpr := app.Config.Init.TokenExpr
	if tokenExpr == "" {
		tokenExpr = "const String.fromEnvironment('APPSTORYS_API_TOKEN')"
	}
	call := fmt.Sprintf("AppStorys.initialize(token: %s);", tokenExpr)

	return []suggest.Suggestion{{
		ID:         suggest.StableID("init", inv.MainFile, "main-init-call"),
		Kind:       "init",
		File:       inv.MainFile,
		InsertAt:   at,
		Snippet:    "\n  " + call,
		Reason:     "Add the AppStorys init call to main(), before runApp",
		Confidence: 0.85,
		Requires:   []string{"add appstorys_sdk_3_0 to pubspec.yaml and enable Android core library desugaring (not yet automated)"},
	}}, nil
}
