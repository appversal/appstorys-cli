package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/patch"
	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

// newIntegratePlacementCmd inserts one inline placement (stories,
// widget, reels or milestone, in the detected platform's own syntax) at
// a location the developer names explicitly. Per the spec, placements
// are product decisions the CLI never guesses at.
func newIntegratePlacementCmd() *cobra.Command {
	var (
		kind     string
		position string
		at       string
		apply    bool
	)

	cmd := &cobra.Command{
		Use:   "placement",
		Short: "Insert an inline placement call at a specific location",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			file, line, err := parseFileLine(at)
			if err != nil {
				return &CLIError{Code: ExitUsageOrConfig, Err: err}
			}

			supported, err := detectSupported(app, nil)
			if err != nil {
				return err
			}
			if len(supported) != 1 {
				return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf(
					"%d platforms detected under %s; point --root at the one project you mean", len(supported), app.Root)}
			}
			body, err := placementSnippet(supported[0].Platform, kind, position)
			if err != nil {
				return &CLIError{Code: ExitUsageOrConfig, Err: err}
			}

			data, exists := readTarget(app.Root, file)
			if !exists {
				return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("%s not found", file)}
			}
			insertAt, err := lineStartOffset(data, line)
			if err != nil {
				return &CLIError{Code: ExitUsageOrConfig, Err: err}
			}

			s := suggest.Suggestion{
				ID:         suggest.StableID("placement", file, fmt.Sprintf("%d:%s:%s", line, kind, position)),
				Kind:       "placement",
				File:       file,
				InsertAt:   insertAt,
				Snippet:    "    " + body + "\n",
				Reason:     fmt.Sprintf("Insert %s placement at %s:%d", kind, file, line),
				Confidence: 1, // developer-directed, not detected
			}

			if !apply {
				fmt.Fprintf(app.Out, "# %s\n", s.Reason)
				fmt.Fprint(app.Out, patch.Render(toEdit(s, false), data))
				return nil
			}

			clean, err := patch.GitClean(app.Root)
			if err != nil {
				return &CLIError{Code: ExitRefused, Err: fmt.Errorf("checking git status: %w", err)}
			}
			if !clean {
				return &CLIError{Code: ExitRefused, Err: fmt.Errorf("git working tree is not clean; commit or stash before --apply")}
			}
			if err := patch.Apply(app.Root, toEdit(s, false)); err != nil {
				return err
			}
			fmt.Fprintln(app.Out, "Applied.")
			return nil
		},
	}

	cmd.Flags().StringVar(&kind, "kind", "", "placement kind: widget|stories|reels|milestone (reels/milestone: android only)")
	cmd.Flags().StringVar(&position, "position", "", "widget position (required for widget/milestone)")
	cmd.Flags().StringVar(&at, "at", "", "location to insert before, as file:line")
	cmd.Flags().BoolVar(&apply, "apply", false, "write the change (requires a clean git tree)")
	_ = cmd.MarkFlagRequired("kind")
	_ = cmd.MarkFlagRequired("at")
	return cmd
}

// placementSnippet returns one inline placement in the detected
// platform's own syntax: a Compose call (Android), a JSX element (React
// Native) or a list element with a trailing comma (Flutter, where
// placements sit inside a children: [...] list).
func placementSnippet(platform project.Platform, kind, position string) (string, error) {
	needPosition := func() error {
		if position == "" {
			return fmt.Errorf("--position is required for --kind %s", kind)
		}
		return nil
	}

	switch platform {
	case project.Android:
		switch kind {
		case "widget", "milestone":
			if err := needPosition(); err != nil {
				return "", err
			}
			call := map[string]string{"widget": "Widget", "milestone": "Milestone"}[kind]
			return fmt.Sprintf("%s(position = %q)", call, position), nil
		case "stories":
			return "Stories()", nil
		case "reels":
			return "Reels()", nil
		}
	case project.ReactNative:
		switch kind {
		case "widget":
			if err := needPosition(); err != nil {
				return "", err
			}
			return fmt.Sprintf("<AppStorys.Widgets position=%q />", position), nil
		case "stories":
			return "<AppStorys.Stories />", nil
		}
	case project.Flutter:
		switch kind {
		case "widget":
			if err := needPosition(); err != nil {
				return "", err
			}
			return fmt.Sprintf("AppStorys.widgets(position: %q),", position), nil
		case "stories":
			return "AppStorys.stories(),", nil
		}
	}
	return "", fmt.Errorf("--kind %q isn't available for %s (android: widget, milestone, stories, reels; react-native and flutter: widget, stories)", kind, platform)
}

func parseFileLine(at string) (file string, line int, err error) {
	idx := strings.LastIndex(at, ":")
	if idx < 0 {
		return "", 0, fmt.Errorf("--at must be file:line, got %q", at)
	}
	file = at[:idx]
	line, err = strconv.Atoi(at[idx+1:])
	if err != nil || line < 1 {
		return "", 0, fmt.Errorf("--at must be file:line with a positive line number, got %q", at)
	}
	return file, line, nil
}

func lineStartOffset(data []byte, line int) (int, error) {
	if line == 1 {
		return 0, nil
	}
	count := 1
	for i, b := range data {
		if b == '\n' {
			count++
			if count == line {
				return i + 1, nil
			}
		}
	}
	return 0, fmt.Errorf("file has fewer than %d lines", line)
}
