package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/androidxml"
	"github.com/appversal/appstorys-cli/internal/lang/dart"
	"github.com/appversal/appstorys-cli/internal/lang/kotlin"
	"github.com/appversal/appstorys-cli/internal/lang/ts"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

// scanSupportedPlatforms is deliberately narrower than project.AllPlatforms:
// extraction doesn't exist for iOS yet (the SDK is binary-only; symbols
// would need to come from a .swiftinterface file, not available
// locally).
var scanSupportedPlatforms = []project.Platform{project.Android, project.Flutter, project.ReactNative}

// languageFor returns the CallExpr walker and LiteralReader for a
// supported platform, plus an optional extraFromTree hook for concepts
// that can't be expressed as a call/constructor match (e.g. React
// Native's appstorys= tag attribute, which can appear on any element).
// It lives here, not on project.Adapter, to avoid the import cycle
// documented in internal/project/adapter.go.
func languageFor(p project.Platform) (lang *sitter.Language, walk func(*sitter.Tree, []byte) []extract.CallExpr, lr extract.LiteralReader, fileExts []string, extraFromTree func(*sitter.Tree, []byte, string) []extract.CallSite, ok bool) {
	switch p {
	case project.Android:
		return kotlin.Language(), kotlin.Extract, kotlin.LiteralReader{}, []string{".kt"}, nil, true
	case project.Flutter:
		return dart.Language(), dart.Extract, dart.LiteralReader{}, []string{".dart"}, nil, true
	case project.ReactNative:
		return ts.Language(), ts.Extract, ts.LiteralReader{}, []string{".ts", ".tsx", ".js", ".jsx"}, ts.ScanTags, true
	default:
		return nil, nil, nil, nil, nil, false
	}
}

// detectSupported detects the project's platform(s) and narrows them to
// the ones this phase can extract from. restrictTo further narrows
// detection (e.g. from --platform); nil means detect anything the
// config or the project itself indicates.
func detectSupported(app *App, restrictTo []project.Platform) ([]project.ProjectInfo, error) {
	infos, err := project.Detect(app.Root, restrictTo)
	if err != nil {
		return nil, err
	}

	var supported []project.ProjectInfo
	for _, info := range infos {
		if slices.Contains(scanSupportedPlatforms, info.Platform) {
			supported = append(supported, info)
		}
	}
	if len(supported) == 0 {
		if len(infos) > 0 {
			return nil, &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf(
				"detected platform(s) under %s aren't supported for scanning yet (iOS extraction isn't implemented)", app.Root)}
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
	lang, walk, lr, fileExts, extraFromTree, ok := languageFor(info.Platform)
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
		if !slices.Contains(fileExts, filepath.Ext(rel)) {
			if info.Platform == project.Android && filepath.Ext(rel) == ".xml" {
				xmlSites, err := scanAndroidLayoutXML(app, rel)
				if err != nil {
					return nil, err
				}
				sites = append(sites, xmlSites...)
			}
			continue
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
		if extraFromTree != nil {
			sites = append(sites, extraFromTree(tree, data, rel)...)
		}
		tree.Close()
	}
	return sites, nil
}

// scanAndroidLayoutXML scans one Android layout XML file for
// OverlayLayoutView/WidgetView, the two layout-XML symbols the Kotlin
// tree-sitter walk doesn't see.
func scanAndroidLayoutXML(app *App, rel string) ([]extract.CallSite, error) {
	data, err := os.ReadFile(filepath.Join(app.Root, rel))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", rel, err)
	}
	sites, err := androidxml.Scan(rel, data)
	if err != nil {
		return nil, err
	}
	return sites, nil
}
