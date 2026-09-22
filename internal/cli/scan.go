package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/dart"
	"github.com/appversal/appstorys-cli/internal/lang/kotlin"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

// scanSupportedPlatforms is deliberately narrower than project.AllPlatforms:
// extraction only exists for Android and Flutter this phase (React
// Native and iOS extraction is Phase 2).
var scanSupportedPlatforms = []project.Platform{project.Android, project.Flutter}

// languageFor returns the CallExpr walker and LiteralReader for a
// supported platform. It lives here, not on project.Adapter, to avoid
// the import cycle documented in internal/project/adapter.go.
func languageFor(p project.Platform) (lang *sitter.Language, walk func(*sitter.Tree, []byte) []extract.CallExpr, lr extract.LiteralReader, fileExt string, ok bool) {
	switch p {
	case project.Android:
		return kotlin.Language(), kotlin.Extract, kotlin.LiteralReader{}, ".kt", true
	case project.Flutter:
		return dart.Language(), dart.Extract, dart.LiteralReader{}, ".dart", true
	default:
		return nil, nil, nil, "", false
	}
}

// detectSupported detects the project's platform(s) and narrows them to
// the ones this phase can extract from (Android, Flutter). restrictTo
// further narrows detection (e.g. from --platform); nil means detect
// anything the config or the project itself indicates.
func detectSupported(app *App, restrictTo []project.Platform) ([]project.ProjectInfo, error) {
	infos, err := project.Detect(app.Root, restrictTo)
	if err != nil {
		return nil, err
	}

	var supported []project.ProjectInfo
	for _, info := range infos {
		for _, p := range scanSupportedPlatforms {
			if info.Platform == p {
				supported = append(supported, info)
				break
			}
		}
	}
	if len(supported) == 0 {
		if len(infos) > 0 {
			return nil, &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf(
				"detected platform(s) under %s aren't supported for scanning yet (Phase 1 covers android and flutter only)", app.Root)}
		}
		return nil, &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf("no supported platform detected under %s", app.Root)}
	}
	return supported, nil
}

// scanProject parses every source file for each detected, supported
// platform and returns every matched CallSite, flattened across
// platforms.
func scanProject(app *App, restrictTo []project.Platform) ([]extract.CallSite, []project.Platform, error) {
	supported, err := detectSupported(app, restrictTo)
	if err != nil {
		return nil, nil, err
	}

	var allSites []extract.CallSite
	var platforms []project.Platform
	for _, info := range supported {
		sites, err := scanPlatform(app, info)
		if err != nil {
			return nil, nil, err
		}
		allSites = append(allSites, sites...)
		platforms = append(platforms, info.Platform)
	}
	return allSites, platforms, nil
}

func scanPlatform(app *App, info project.ProjectInfo) ([]extract.CallSite, error) {
	lang, walk, lr, fileExt, ok := languageFor(info.Platform)
	if !ok {
		return nil, fmt.Errorf("scan: no extractor for platform %q", info.Platform)
	}

	adapter, _ := project.ByPlatform(info.Platform)
	files, err := adapter.Files(app.Root)
	if err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}

	sm, err := symbols.LoadPlatform(string(info.Platform))
	if err != nil {
		return nil, &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("loading %s symbol map: %w", info.Platform, err)}
	}

	pool, err := parse.NewPool(lang, 0)
	if err != nil {
		return nil, fmt.Errorf("starting parser pool: %w", err)
	}
	defer pool.Close()

	opts := extract.Options{ExtraReceivers: app.Config.Receivers}

	var sites []extract.CallSite
	for _, rel := range files {
		if filepath.Ext(rel) != fileExt {
			continue // e.g. skip Android's res/layout/*.xml here; no XML adapter yet
		}
		data, err := os.ReadFile(filepath.Join(app.Root, rel))
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", rel, err)
		}
		tree, err := pool.Parse(context.Background(), data)
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", rel, err)
		}
		calls := walk(tree, data)
		sites = append(sites, extract.Match(calls, rel, data, info.Platform, sm, lr, opts)...)
		tree.Close()
	}
	return sites, nil
}
