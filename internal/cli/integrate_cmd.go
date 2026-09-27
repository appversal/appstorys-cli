package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/output"
	"github.com/appversal/appstorys-cli/internal/patch"
	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

func newIntegrateCmd() *cobra.Command {
	var (
		only          string
		minConfidence float64
		suggestionID  string
		showDiff      bool
		apply         bool
		format        string
	)

	cmd := &cobra.Command{
		Use:   "integrate",
		Short: "Suggest and apply overlay-host and screen-tracking call sites",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			supported, err := detectSupported(app, nil)
			if err != nil {
				return err
			}

			var suggestions []suggest.Suggestion
			for _, info := range supported {
				switch info.Platform {
				case project.Android:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					s, err := planIntegrateAndroid(app, sites)
					if err != nil {
						return err
					}
					suggestions = append(suggestions, s...)
				case project.ReactNative:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					s, err := planIntegrateReactNative(app, sites)
					if err != nil {
						return err
					}
					suggestions = append(suggestions, s...)
				case project.Flutter:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					s, err := planIntegrateFlutter(app, sites)
					if err != nil {
						return err
					}
					suggestions = append(suggestions, s...)
				default:
					return &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf(
						"integrate doesn't support %s yet (only android, react-native and flutter are implemented so far)", info.Platform)}
				}
			}

			suggestions, err = filterSuggestions(suggestions, only, minConfidence, suggestionID)
			if err != nil {
				return &CLIError{Code: ExitUsageOrConfig, Err: err}
			}

			// JSON consumers always get JSON, including for "nothing to do";
			// the human sentence is only for the other formats.
			if format == "json" {
				return output.WriteJSON(app.Out, suggestions)
			}
			if len(suggestions) == 0 {
				fmt.Fprintln(app.Out, "No integration gaps found (or none matched the given filters).")
				return nil
			}

			switch {
			case apply:
				clean, err := patch.GitClean(app.Root)
				if err != nil {
					return &CLIError{Code: ExitRefused, Err: fmt.Errorf("checking git status: %w", err)}
				}
				if !clean {
					return &CLIError{Code: ExitRefused, Err: fmt.Errorf("git working tree is not clean; commit or stash before --apply")}
				}
				if err := applySuggestions(app.Root, suggestions); err != nil {
					return err
				}
				fmt.Fprintf(app.Out, "Applied %d change(s). Screen names are campaign-targeting keys: confirm each one matches what the dashboard expects.\n", len(suggestions))
				return nil
			case showDiff:
				seenID := map[string]bool{}
				for _, s := range suggestions {
					if !seenID[s.ID] {
						seenID[s.ID] = true
						fmt.Fprintf(app.Out, "# [%s] %s\n", s.ID, s.Reason)
					}
					original, exists := readTarget(app.Root, s.File)
					fmt.Fprint(app.Out, patch.Render(toEdit(s, !exists), original))
					fmt.Fprintln(app.Out)
				}
				return nil
			default:
				return writeSuggestionReport(app, suggestions)
			}
		},
	}

	cmd.Flags().StringVar(&only, "only", "", "restrict to overlay|screen")
	cmd.Flags().Float64Var(&minConfidence, "min-confidence", 0, "drop suggestions below this confidence (0..1)")
	cmd.Flags().StringVar(&suggestionID, "suggestion", "", "act on exactly one suggestion by ID")
	cmd.Flags().BoolVar(&showDiff, "diff", false, "show a unified diff for each suggestion")
	cmd.Flags().BoolVar(&apply, "apply", false, "write the changes (requires a clean git tree)")
	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json")

	cmd.AddCommand(newIntegratePlacementCmd())
	return cmd
}

// filterSuggestions applies --only/--min-confidence/--suggestion.
// --suggestion returns every suggestion sharing that ID, not just the
// first: a paired edit (e.g. integrate's <AppStorys.Screen> open/close
// tags) shares one ID on purpose, since applying just one half would
// leave broken JSX.
func filterSuggestions(suggestions []suggest.Suggestion, only string, minConfidence float64, id string) ([]suggest.Suggestion, error) {
	if id != "" {
		var matched []suggest.Suggestion
		for _, s := range suggestions {
			if s.ID == id {
				matched = append(matched, s)
			}
		}
		if len(matched) == 0 {
			return nil, fmt.Errorf("no suggestion with ID %q (it may be stale; re-run without --suggestion to list current ones)", id)
		}
		return matched, nil
	}

	var kind string
	switch only {
	case "":
		// no kind filter
	case "overlay":
		kind = "overlay-host"
	case "screen":
		kind = "screen"
	default:
		return nil, fmt.Errorf("--only must be overlay or screen, got %q", only)
	}

	out := suggestions[:0:0]
	for _, s := range suggestions {
		if kind != "" && s.Kind != kind {
			continue
		}
		if s.Confidence < minConfidence {
			continue
		}
		out = append(out, s)
	}
	return out, nil
}

var suggestionHeaders = []string{"id", "kind", "file", "confidence", "reason"}

func writeSuggestionReport(app *App, suggestions []suggest.Suggestion) error {
	seenID := map[string]bool{}
	rows := make([][]string, 0, len(suggestions))
	for _, s := range suggestions {
		if seenID[s.ID] {
			continue // paired edit (e.g. an open/close JSX tag pair): one row per logical suggestion
		}
		seenID[s.ID] = true
		rows = append(rows, []string{s.ID, s.Kind, s.File, fmt.Sprintf("%.2f", s.Confidence), s.Reason})
	}
	if err := output.WriteTable(app.Out, suggestionHeaders, rows); err != nil {
		return err
	}
	fmt.Fprintln(app.Out, "\nRun with --diff to see the change, or --suggestion <id> --apply to apply just one.")
	return nil
}
