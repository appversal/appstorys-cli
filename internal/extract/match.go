package extract

import (
	"slices"
	"strings"

	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

// Options configures Match beyond the symbol map itself.
type Options struct {
	// ExtraReceivers are configured aliases for the platform's SDK
	// singleton (e.g. "App.appStorys" from .appstorys-cli.yaml's
	// `receivers:` list), accepted anywhere one of the symbol map's own
	// Receivers would be.
	ExtraReceivers []string
}

// Match walks calls, matches each against m's symbols, and returns the
// resulting CallSites. A call that matches no symbol is skipped (it's
// not an AppStorys call the CLI recognizes). file and platform are
// stamped onto every resulting CallSite.
func Match(calls []CallExpr, file string, src []byte, platform project.Platform, m *symbols.Map, lr LiteralReader, opts Options) []CallSite {
	var sites []CallSite
	for _, call := range calls {
		method := call.Method()
		if method == "" {
			continue
		}
		receiverPath := call.ReceiverPath()

		for _, sym := range m.Symbols {
			qualified := sym.Call
			if qualified == "" {
				qualified = sym.Constructor
			}
			if qualified == "" {
				continue
			}
			expectedMethod, expectedReceiver := splitQualified(qualified)
			if method != expectedMethod {
				continue
			}
			if !receiverMatches(expectedReceiver, receiverPath, m.Receivers, opts.ExtraReceivers) {
				continue
			}

			site := CallSite{
				Kind:      sym.Concept,
				File:      file,
				Enclosing: call.Enclosing,
				Via:       qualified,
				Platform:  platform,
			}
			if call.Node != nil {
				pos := call.Node.StartPosition()
				site.Line = int(pos.Row) + 1
				site.Col = int(pos.Column) + 1
			}

			if sym.NameArg != nil {
				if arg, ok := FindArg(call.Args, sym.NameArg); ok {
					if text, isLiteral := lr.StringLiteral(arg.Node, src); isLiteral {
						site.Name = text
					} else {
						site.Dynamic = true
					}
				} else {
					site.Dynamic = true
				}
			}

			if sym.PositionArg != nil {
				if arg, ok := FindArg(call.Args, sym.PositionArg); ok {
					if text, isLiteral := lr.StringLiteral(arg.Node, src); isLiteral {
						site.Position = text
					}
				}
			}
			if sym.PositionsArg != nil {
				if arg, ok := FindArg(call.Args, sym.PositionsArg); ok {
					items := lr.ListLiteral(arg.Node, src)
					if items != nil {
						site.Position = strings.Join(items, ",")
					}
				}
			}

			if sym.PropsArg != nil {
				if arg, ok := FindArg(call.Args, sym.PropsArg); ok {
					entries := lr.MapEntries(arg.Node, src)
					if entries != nil {
						site.Properties = make(map[string]PropType, len(entries))
						for _, e := range entries {
							site.Properties[e.Key] = lr.PropType(e.Value)
						}
					}
				}
			}

			sites = append(sites, site)
			break // first matching symbol wins; symbol lists don't overlap in practice
		}
	}
	return sites
}

// splitQualified splits "Receiver.method" into ("method", "Receiver"),
// or "method" into ("method", "") for a bare call/constructor.
func splitQualified(qualified string) (method, receiver string) {
	if idx := strings.LastIndex(qualified, "."); idx >= 0 {
		return qualified[idx+1:], qualified[:idx]
	}
	return qualified, ""
}

// receiverMatches decides whether a call's actual receiver path
// satisfies a symbol's expected receiver.
//
//   - expected == "" requires a bare call (actual == "").
//   - expected == actual is always a match.
//   - if expected is one of the symbol map's canonical receivers (e.g.
//     "AppStorys"), the actual path may also be expected+".getInstance"
//     (the SDK's `getInstance()` accessor) or any configured alias
//     (e.g. "App.appStorys") — see the spec's "Receivers and wrappers
//     to handle" column. A receiver that isn't canonical (e.g.
//     "Modifier" for Android's tag call) gets no aliasing: only an
//     exact match counts.
func receiverMatches(expected, actual string, canonicalReceivers, aliasReceivers []string) bool {
	if expected == actual {
		return true
	}
	if expected == "" {
		return false
	}
	if !slices.Contains(canonicalReceivers, expected) {
		return false
	}
	if actual == expected+".getInstance" {
		return true
	}
	return slices.Contains(aliasReceivers, actual)
}
