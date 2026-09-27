package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/androidxml"
	"github.com/appversal/appstorys-cli/internal/lang/kotlin"
	"github.com/appversal/appstorys-cli/internal/parse"
	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// planIntegrateAndroid proposes overlay-host and screen-tracking
// suggestions for Android.
//
// Compose NavHost destinations get both overlay-host and screen
// suggestions ("High" confidence per spec — screen state is global and
// there's a single host). Activities declared in the manifest that
// aren't tracked get a screen suggestion ("Medium" confidence, reusing
// doctor's discovery); those missing an overlay host get one too, but
// only when a setContent { ... } block can be found somewhere in the
// Activity's own class body (any method, not just onCreate — it's
// common to call setContent from a helper like setupContent() rather
// than directly) — overlayElements() has to run inside the composable
// content lambda, so an Activity where setContent isn't found there (XML
// layouts instead, a base class's onCreate, or something this pass's
// detection missed) is left to doctor's report rather than risk
// inserting a call that looks plausible but doesn't actually run in the
// right place.
//
// Event suggestions (purchase flows, auth callbacks, ranked interaction
// handlers) and `--interactive` mode aren't implemented — deferred as
// separate scope.
func planIntegrateAndroid(app *App, sites []extract.CallSite) ([]suggest.Suggestion, error) {
	compose, err := planComposeNavHosts(app, sites)
	if err != nil {
		return nil, err
	}
	views, err := planViewsScreens(app, sites)
	if err != nil {
		return nil, err
	}
	overlay, err := planViewsOverlay(app, sites)
	if err != nil {
		return nil, err
	}
	suggestions := append(compose, views...)
	return append(suggestions, overlay...), nil
}

func planComposeNavHosts(app *App, sites []extract.CallSite) ([]suggest.Suggestion, error) {
	adapter, _ := project.ByPlatform(project.Android)
	files, err := adapter.Files(app.Root)
	if err != nil {
		return nil, fmt.Errorf("listing files: %w", err)
	}

	pool, err := parse.NewPool(kotlin.Language(), 0)
	if err != nil {
		return nil, err
	}
	defer pool.Close()

	var suggestions []suggest.Suggestion
	for _, rel := range files {
		if filepath.Ext(rel) != ".kt" {
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
		hosts := kotlin.ComposeNavHosts(tree, data)
		tree.Close()
		if len(hosts) == 0 {
			continue
		}

		fileHasOverlay := fileHasKind(sites, rel, "overlay-host")

		for _, h := range hosts {
			if !fileHasOverlay && h.EnclosingBodyInsertAt > 0 {
				suggestions = append(suggestions, suggest.Suggestion{
					ID:         suggest.StableID("overlay-host", rel, h.EnclosingFunc),
					Kind:       "overlay-host",
					File:       rel,
					InsertAt:   h.EnclosingBodyInsertAt,
					Snippet:    "\n    overlayElements()",
					Anchor:     h.EnclosingFunc,
					Reason:     fmt.Sprintf("Add the overlay host once in %s, the composable enclosing NavHost", h.EnclosingFunc),
					Confidence: 0.9,
				})
				fileHasOverlay = true // one per file is enough, even with multiple NavHosts
			}

			for _, d := range h.Destinations {
				if d.Dynamic {
					continue // can't safely register a name we can't read
				}
				if lineRangeHasKind(sites, rel, "screen", d.StartLine, d.EndLine) {
					continue
				}
				name := deriveScreenName(d.Route)
				suggestions = append(suggestions, suggest.Suggestion{
					ID:       suggest.StableID("screen", rel, d.Route),
					Kind:     "screen",
					File:     rel,
					InsertAt: d.BodyInsertAt,
					Snippet: fmt.Sprintf("\n            LaunchedEffect(Unit) {\n                AppStorys.getScreenCampaigns(%q)\n            }",
						name),
					Anchor:     d.Route,
					Reason:     fmt.Sprintf("Track destination %q as screen %q", d.Route, name),
					Confidence: 0.9,
					Requires:   []string{"confirm screen name"},
				})
			}
		}
	}
	return suggestions, nil
}

func planViewsScreens(app *App, sites []extract.CallSite) ([]suggest.Suggestion, error) {
	manifestPath, err := findManifest(app.Root)
	if err != nil {
		return nil, err
	}
	if manifestPath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(app.Root, manifestPath))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", manifestPath, err)
	}
	manifest, err := androidxml.ParseManifest(data)
	if err != nil {
		return nil, err
	}
	classes, err := allKotlinClasses(app)
	if err != nil {
		return nil, err
	}

	var suggestions []suggest.Suggestion
	for _, act := range manifest.Activities {
		simple := androidxml.SimpleName(act.Name)
		if simple == "" {
			continue
		}
		var classInfo *kotlin.ClassInfo
		for i := range classes {
			if classes[i].Name == simple {
				classInfo = &classes[i]
				break
			}
		}
		if classInfo == nil {
			continue // doctor already reports this as heuristic-missing
		}
		if fileHasKind(sites, classInfo.File, "screen") {
			continue
		}
		if hostsComposeNav, err := fileHasComposeNavHost(app.Root, classInfo.File); err != nil {
			return nil, err
		} else if hostsComposeNav {
			// This Activity just hosts a NavHost; its destinations are
			// the real screens and already got their own suggestions
			// from planComposeNavHosts. The Activity itself isn't one.
			continue
		}

		s, err := screenCallSuggestion(app.Root, *classInfo, simple)
		if err != nil {
			return nil, err
		}
		suggestions = append(suggestions, s)
	}
	return suggestions, nil
}

func screenCallSuggestion(root string, classInfo kotlin.ClassInfo, activityName string) (suggest.Suggestion, error) {
	data, err := os.ReadFile(filepath.Join(root, classInfo.File))
	if err != nil {
		return suggest.Suggestion{}, fmt.Errorf("reading %s: %w", classInfo.File, err)
	}
	pool, err := parse.NewPool(kotlin.Language(), 1)
	if err != nil {
		return suggest.Suggestion{}, err
	}
	defer pool.Close()
	tree, err := pool.Parse(context.Background(), data)
	if err != nil {
		return suggest.Suggestion{}, fmt.Errorf("parsing %s: %w", classInfo.File, err)
	}
	defer tree.Close()

	bodyAt, methods, found := kotlin.LocateClass(tree, data, classInfo.Name)
	if !found {
		return suggest.Suggestion{}, fmt.Errorf("could not re-locate class %s in %s", classInfo.Name, classInfo.File)
	}

	name := deriveScreenName(activityName)
	call := fmt.Sprintf("AppStorys.getScreenCampaigns(%q)", name)

	if at, ok := methods["onResume"]; ok {
		return suggest.Suggestion{
			ID:         suggest.StableID("screen", classInfo.File, "onResume"),
			Kind:       "screen",
			File:       classInfo.File,
			InsertAt:   at,
			Snippet:    "\n        " + call,
			Anchor:     activityName,
			Reason:     fmt.Sprintf("Track %s as screen %q in onResume", classInfo.Name, name),
			Confidence: 0.6,
			Requires:   []string{"confirm screen name"},
		}, nil
	}

	snippet := fmt.Sprintf("\n    override fun onResume() {\n        super.onResume()\n        %s\n    }\n", call)
	return suggest.Suggestion{
		ID:         suggest.StableID("screen", classInfo.File, "new-onResume"),
		Kind:       "screen",
		File:       classInfo.File,
		InsertAt:   bodyAt,
		Snippet:    snippet,
		Anchor:     activityName,
		Reason:     fmt.Sprintf("%s has no onResume; adding one to track screen %q", classInfo.Name, name),
		Confidence: 0.55,
		Requires:   []string{"confirm screen name"},
	}, nil
}

// fileHasComposeNavHost reports whether file contains a NavHost(...) {
// ... } block, used to tell a plain Activity apart from one that just
// hosts Compose Navigation (whose real screens are its destinations,
// not the Activity itself).
// planViewsOverlay proposes overlay-host suggestions for manifest
// Activities missing one, by finding setContent { ... } anywhere in
// their class body. Activities that host a NavHost are skipped — their
// overlay host suggestion already came from planComposeNavHosts,
// targeting the composable that encloses the NavHost, not the
// Activity's setContent; proposing both would risk two
// overlayElements() calls.
func planViewsOverlay(app *App, sites []extract.CallSite) ([]suggest.Suggestion, error) {
	manifestPath, err := findManifest(app.Root)
	if err != nil {
		return nil, err
	}
	if manifestPath == "" {
		return nil, nil
	}
	data, err := os.ReadFile(filepath.Join(app.Root, manifestPath))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", manifestPath, err)
	}
	manifest, err := androidxml.ParseManifest(data)
	if err != nil {
		return nil, err
	}
	classes, err := allKotlinClasses(app)
	if err != nil {
		return nil, err
	}

	var suggestions []suggest.Suggestion
	for _, act := range manifest.Activities {
		simple := androidxml.SimpleName(act.Name)
		if simple == "" {
			continue
		}
		var classInfo *kotlin.ClassInfo
		for i := range classes {
			if classes[i].Name == simple {
				classInfo = &classes[i]
				break
			}
		}
		if classInfo == nil {
			continue
		}
		if fileHasKind(sites, classInfo.File, "overlay-host") {
			continue
		}
		hostsComposeNav, err := fileHasComposeNavHost(app.Root, classInfo.File)
		if err != nil {
			return nil, err
		}
		if hostsComposeNav {
			continue
		}

		s, err := overlayHostSuggestion(app.Root, *classInfo)
		if err != nil {
			return nil, err
		}
		if s != nil {
			suggestions = append(suggestions, *s)
		}
	}
	return suggestions, nil
}

func overlayHostSuggestion(root string, classInfo kotlin.ClassInfo) (*suggest.Suggestion, error) {
	data, err := os.ReadFile(filepath.Join(root, classInfo.File))
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", classInfo.File, err)
	}
	pool, err := parse.NewPool(kotlin.Language(), 1)
	if err != nil {
		return nil, err
	}
	defer pool.Close()
	tree, err := pool.Parse(context.Background(), data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", classInfo.File, err)
	}
	defer tree.Close()

	at, found := kotlin.LocateClassAnyMethodBlock(tree, data, classInfo.Name, "setContent")
	if !found {
		return nil, nil // can't safely place it; leave to doctor's report
	}

	return &suggest.Suggestion{
		ID:         suggest.StableID("overlay-host", classInfo.File, "setContent"),
		Kind:       "overlay-host",
		File:       classInfo.File,
		InsertAt:   at,
		Snippet:    blockSnippet(data, at, "\n            overlayElements()"),
		Anchor:     classInfo.Name,
		Reason:     fmt.Sprintf("Add the overlay host inside %s's setContent", classInfo.Name),
		Confidence: 0.6,
	}, nil
}

func fileHasComposeNavHost(root, file string) (bool, error) {
	data, err := os.ReadFile(filepath.Join(root, file))
	if err != nil {
		return false, fmt.Errorf("reading %s: %w", file, err)
	}
	pool, err := parse.NewPool(kotlin.Language(), 1)
	if err != nil {
		return false, err
	}
	defer pool.Close()
	tree, err := pool.Parse(context.Background(), data)
	if err != nil {
		return false, fmt.Errorf("parsing %s: %w", file, err)
	}
	defer tree.Close()
	return len(kotlin.ComposeNavHosts(tree, data)) > 0, nil
}

func fileHasKind(sites []extract.CallSite, file, kind string) bool {
	for _, s := range sites {
		if s.File == file && s.Kind == kind {
			return true
		}
	}
	return false
}

func lineRangeHasKind(sites []extract.CallSite, file, kind string, start, end int) bool {
	for _, s := range sites {
		if s.File == file && s.Kind == kind && s.Line >= start && s.Line <= end {
			return true
		}
	}
	return false
}

var screenNameSuffixes = []string{"Activity", "Fragment", "ViewController", "Screen"}

// deriveScreenName proposes a screen name from a route or class name,
// per the spec's own example ("CheckoutViewController -> Checkout"): it
// strips a trailing component-kind suffix, then splits camelCase/
// snake_case/kebab-case/slash-separated words and title-cases each. The
// result is always shown for confirmation before it's ever applied — no
// SDK call site's existing name is ever touched by this CLI.
func deriveScreenName(raw string) string {
	for _, suf := range screenNameSuffixes {
		if strings.HasSuffix(raw, suf) && len(raw) > len(suf) {
			raw = strings.TrimSuffix(raw, suf)
			break
		}
	}

	var words []string
	var cur strings.Builder
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, cur.String())
			cur.Reset()
		}
	}
	runes := []rune(raw)
	for i, r := range runes {
		switch {
		case r == '_' || r == '-' || r == '/':
			flush()
		case r >= 'A' && r <= 'Z' && i > 0 && !(runes[i-1] >= 'A' && runes[i-1] <= 'Z'):
			flush()
			cur.WriteRune(r)
		default:
			cur.WriteRune(r)
		}
	}
	flush()

	for i, w := range words {
		if w == "" {
			continue
		}
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}
