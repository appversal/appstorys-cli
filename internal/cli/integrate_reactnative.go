package cli

import (
	"fmt"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/lang/ts"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// planIntegrateReactNative proposes wrapping an untracked navigator
// screen's returned JSX in <AppStorys.Screen name="...">, which — per
// the spec's RN row — both tracks the screen and provides its overlay
// host in one wrapper; no separate overlay-host suggestion is needed
// the way Android needs one.
//
// A wrap is two insertions (the opening and closing tag), which the
// Suggestion/patch model represents as two Suggestion values sharing
// one ID rather than extending that model for one platform's needs —
// filterSuggestions and the report/diff output already treat
// same-ID suggestions as one logical, always-apply-together unit.
//
// Only routes whose component has a simple, single-JSX-expression
// return (ts.Components' ReturnJSXStart/End) are wrapped — a
// conditional return, multiple returns, or a fragment isn't safe to
// wrap blindly, so those are left to doctor's report instead.
func planIntegrateReactNative(app *App, sites []extract.CallSite) ([]suggest.Suggestion, error) {
	inv, err := reactNativeInventory(app)
	if err != nil {
		return nil, err
	}

	var suggestions []suggest.Suggestion
	for _, route := range inv.Routes {
		var comp *ts.ComponentInfo
		for i := range inv.Components {
			if inv.Components[i].Name == route.ComponentName {
				comp = &inv.Components[i]
				break
			}
		}
		if comp == nil {
			continue // doctor already reports this as heuristic-missing
		}
		if fileHasKind(sites, comp.File, "screen") {
			continue // already tracked
		}
		if comp.ReturnJSXStart == 0 && comp.ReturnJSXEnd == 0 {
			continue // return shape too complex to safely wrap
		}

		screenName := route.RouteName
		if screenName == "" {
			screenName = route.ComponentName
		}

		id := suggest.StableID("screen", comp.File, "wrap-screen-"+route.ComponentName)
		reason := fmt.Sprintf("Wrap %s's returned JSX in <AppStorys.Screen name=%q>, which also tracks the screen",
			route.ComponentName, screenName)

		suggestions = append(suggestions,
			suggest.Suggestion{
				ID:         id,
				Kind:       "screen",
				File:       comp.File,
				InsertAt:   comp.ReturnJSXStart,
				Snippet:    fmt.Sprintf("<AppStorys.Screen name=%q>\n", screenName),
				Anchor:     route.ComponentName,
				Reason:     reason,
				Confidence: 0.9,
				Requires:   []string{"confirm screen name"},
			},
			suggest.Suggestion{
				ID:         id,
				Kind:       "screen",
				File:       comp.File,
				InsertAt:   comp.ReturnJSXEnd,
				Snippet:    "\n</AppStorys.Screen>",
				Anchor:     route.ComponentName,
				Reason:     reason,
				Confidence: 0.9,
				Requires:   []string{"confirm screen name"},
			},
		)
	}
	return suggestions, nil
}
