package cli

import (
	"fmt"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/dart"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// planIntegrateFlutter proposes screen-tracking and overlay-host
// suggestions for every MaterialApp(routes: {...}) entry: trackScreen(
// name, context) in the route widget's State.initState (creating one if
// missing, mirroring how planInitAndroid/planInitReactNative create a
// missing hook), and ...AppStorys.overlayElements() inside the screen's
// existing Stack.
//
// Per the spec, wrapping a screen's body in a new Stack when one is
// absent is "diff only, never auto-apply" — rather than half-implement
// that distinction, a screen with no Stack simply gets no overlay-host
// suggestion at all here (see dart.StackChildrenInsertPoint), left to
// doctor's report instead of a suggestion this command can't safely
// apply anyway.
func planIntegrateFlutter(app *App, sites []extract.CallSite) ([]suggest.Suggestion, error) {
	inv, err := flutterInventoryFor(app)
	if err != nil {
		return nil, err
	}

	var suggestions []suggest.Suggestion
	for _, route := range inv.Routes {
		if route.WidgetName == "" {
			continue
		}
		file, _, ok := flutterScreenFile(inv.Classes, route.WidgetName)
		if !ok {
			continue // doctor already reports this as heuristic-missing
		}

		var stateClass string
		for _, c := range inv.Classes {
			if c.Supertype == "State" && c.StateOf == route.WidgetName {
				stateClass = c.Name
				break
			}
		}

		if !fileHasKind(sites, file, "screen") && stateClass != "" {
			s, err := flutterScreenTrackingSuggestion(app.Root, stateClass, file, deriveScreenName(route.WidgetName))
			if err != nil {
				return nil, err
			}
			suggestions = append(suggestions, s)
		}

		if !fileHasKind(sites, file, "overlay-host") {
			s, err := flutterOverlaySuggestion(app.Root, file)
			if err != nil {
				return nil, err
			}
			if s != nil {
				suggestions = append(suggestions, *s)
			}
		}
	}
	return suggestions, nil
}

func flutterScreenTrackingSuggestion(root, stateClass, file, screenName string) (suggest.Suggestion, error) {
	tree, data, closeFn, err := parseDartFile(root, file)
	if err != nil {
		return suggest.Suggestion{}, err
	}
	defer closeFn()

	call := fmt.Sprintf("AppStorys.trackScreen(%q, context);", screenName)

	if at, ok := dart.LocateClassMethod(tree, data, stateClass, "initState"); ok {
		return suggest.Suggestion{
			ID:         suggest.StableID("screen", file, "initState-"+stateClass),
			Kind:       "screen",
			File:       file,
			InsertAt:   at,
			Snippet:    "\n    " + call,
			Anchor:     stateClass,
			Reason:     fmt.Sprintf("Track %s as screen %q in initState", stateClass, screenName),
			Confidence: 0.6,
			Requires:   []string{"confirm screen name"},
		}, nil
	}

	at, ok := dart.LocateClassBody(tree, data, stateClass)
	if !ok {
		return suggest.Suggestion{}, fmt.Errorf("could not re-locate class %s in %s", stateClass, file)
	}
	snippet := fmt.Sprintf("\n  @override\n  void initState() {\n    super.initState();\n    %s\n  }\n", call)
	return suggest.Suggestion{
		ID:         suggest.StableID("screen", file, "new-initState-"+stateClass),
		Kind:       "screen",
		File:       file,
		InsertAt:   at,
		Snippet:    snippet,
		Anchor:     stateClass,
		Reason:     fmt.Sprintf("%s has no initState; adding one to track screen %q", stateClass, screenName),
		Confidence: 0.55,
		Requires:   []string{"confirm screen name"},
	}, nil
}

func flutterOverlaySuggestion(root, file string) (*suggest.Suggestion, error) {
	tree, data, closeFn, err := parseDartFile(root, file)
	if err != nil {
		return nil, err
	}
	defer closeFn()

	at, ok := dart.StackChildrenInsertPoint(tree, data)
	if !ok {
		return nil, nil
	}
	return &suggest.Suggestion{
		ID:         suggest.StableID("overlay-host", file, "stack-children"),
		Kind:       "overlay-host",
		File:       file,
		InsertAt:   at,
		Snippet:    blockSnippet(data, at, "\n        ...AppStorys.overlayElements(),"),
		Reason:     "Add the overlay host inside the screen's Stack",
		Confidence: 0.6,
	}, nil
}
