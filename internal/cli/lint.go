package cli

import (
	"fmt"

	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/rules"
	"github.com/appversal/appstorys-cli/internal/symbols"
)

// lintProject scans every detected, supported platform and runs the
// lint rules against each with its own symbol map (reserved event
// names, merged metadata keys, etc. differ per platform).
func lintProject(app *App, restrictTo []project.Platform) ([]rules.Finding, error) {
	supported, err := detectSupported(app, restrictTo)
	if err != nil {
		return nil, err
	}

	var findings []rules.Finding
	for _, info := range supported {
		sites, err := scanPlatform(app, info)
		if err != nil {
			return nil, err
		}
		sm, err := symbols.LoadPlatform(string(info.Platform))
		if err != nil {
			return nil, &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("loading %s symbol map: %w", info.Platform, err)}
		}
		findings = append(findings, rules.Run(sites, sm, app.Config.Rules)...)
	}
	return findings, nil
}

// severityAtOrAbove reports whether any finding meets or exceeds
// threshold on the spec's info < warning < error ordering.
func severityAtOrAbove(findings []rules.Finding, threshold rules.Severity) bool {
	rank := map[rules.Severity]int{rules.SeverityInfo: 0, rules.SeverityWarning: 1, rules.SeverityError: 2}
	min := rank[threshold]
	for _, f := range findings {
		if rank[f.Severity] >= min {
			return true
		}
	}
	return false
}
