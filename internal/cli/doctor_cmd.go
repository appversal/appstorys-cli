package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/output"
	"github.com/appversal/appstorys-cli/internal/project"
)

func newDoctorCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check init placement, screen tracking and overlay hosts",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			supported, err := detectSupported(app, nil)
			if err != nil {
				return err
			}

			var results []DoctorResult
			for _, info := range supported {
				switch info.Platform {
				case project.Android:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					r, err := doctorAndroid(app, sites)
					if err != nil {
						return err
					}
					results = append(results, r...)
				case project.ReactNative:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					r, err := doctorReactNative(app, sites)
					if err != nil {
						return err
					}
					results = append(results, r...)
				case project.Flutter:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					r, err := doctorFlutter(app, sites)
					if err != nil {
						return err
					}
					results = append(results, r...)
				default:
					return &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf(
						"doctor doesn't support %s yet (only android, react-native and flutter are implemented so far)", info.Platform)}
				}
			}

			if err := writeDoctorResults(app, format, results); err != nil {
				return err
			}
			for _, r := range results {
				if strings.HasPrefix(r.Status, "fail:") {
					return &CLIError{Code: ExitFindings, Err: fmt.Errorf("doctor found failing checks")}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json|md")
	return cmd
}

var doctorHeaders = []string{"check", "screen", "file:line", "tracked as", "overlay", "status"}

func doctorRows(results []DoctorResult) [][]string {
	rows := make([][]string, 0, len(results))
	for _, r := range results {
		loc := "-"
		if r.File != "" && r.File != "-" {
			loc = r.File
			if r.Line > 0 {
				loc += ":" + strconv.Itoa(r.Line)
			}
		}
		trackedAs := r.TrackedAs
		if trackedAs == "" {
			trackedAs = "-"
		}
		overlay := r.Overlay
		if overlay == "" {
			overlay = "-"
		}
		screen := r.Screen
		if screen == "" {
			screen = "-"
		}
		rows = append(rows, []string{r.Check, screen, loc, trackedAs, overlay, r.Status})
	}
	return rows
}

func writeDoctorResults(app *App, format string, results []DoctorResult) error {
	switch format {
	case "json":
		return output.WriteJSON(app.Out, results)
	case "md":
		return output.WriteMarkdownTable(app.Out, doctorHeaders, doctorRows(results))
	case "table", "":
		return output.WriteTable(app.Out, doctorHeaders, doctorRows(results))
	default:
		return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("unknown --format %q", format)}
	}
}
