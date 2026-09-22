package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/output"
	"github.com/appversal/appstorys-cli/internal/rules"
)

func newLintCmd() *cobra.Command {
	var format string

	cmd := &cobra.Command{
		Use:   "lint",
		Short: "Run integration and tracking lint rules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			findings, err := lintProject(app, nil)
			if err != nil {
				return err
			}

			if err := writeFindings(app, format, findings); err != nil {
				return err
			}
			if severityAtOrAbove(findings, rules.SeverityError) {
				return &CLIError{Code: ExitFindings, Err: fmt.Errorf("lint found errors")}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json|sarif|md")
	return cmd
}

func newValidateCmd() *cobra.Command {
	var (
		format string
		failOn string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "CI gate: run lint rules and fail on findings at or above --fail-on",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			threshold := rules.Severity(failOn)
			if threshold != rules.SeverityError && threshold != rules.SeverityWarning {
				return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("--fail-on must be error or warning, got %q", failOn)}
			}

			findings, err := lintProject(app, nil)
			if err != nil {
				return err
			}

			if err := writeFindings(app, format, findings); err != nil {
				return err
			}
			if severityAtOrAbove(findings, threshold) {
				return &CLIError{Code: ExitFindings, Err: fmt.Errorf("validate found findings at or above %s", failOn)}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json|sarif|md")
	cmd.Flags().StringVar(&failOn, "fail-on", "error", "minimum severity that fails the build: error|warning")
	return cmd
}

var lintHeaders = []string{"rule", "severity", "file:line", "message"}

func writeFindings(app *App, format string, findings []rules.Finding) error {
	switch format {
	case "json":
		return output.WriteJSON(app.Out, findings)
	case "sarif":
		results := make([]output.SARIFResult, 0, len(findings))
		for _, f := range findings {
			results = append(results, output.SARIFResult{
				RuleID:  f.RuleID,
				Level:   output.SeverityToSARIFLevel(string(f.Severity)),
				Message: f.Message,
				File:    f.Site.File,
				Line:    f.Site.Line,
				Col:     f.Site.Col,
			})
		}
		return output.WriteSARIF(app.Out, "appstorys-cli", Version, results)
	case "md":
		return output.WriteMarkdownTable(app.Out, lintHeaders, lintRows(findings))
	case "table", "":
		return output.WriteTable(app.Out, lintHeaders, lintRows(findings))
	default:
		return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("unknown --format %q", format)}
	}
}

func lintRows(findings []rules.Finding) [][]string {
	rows := make([][]string, 0, len(findings))
	for _, f := range findings {
		loc := "-"
		if f.Site.File != "" {
			loc = fmt.Sprintf("%s:%d", f.Site.File, f.Site.Line)
		}
		rows = append(rows, []string{f.RuleID, string(f.Severity), loc, f.Message})
	}
	return rows
}
