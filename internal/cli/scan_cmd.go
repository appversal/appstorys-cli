package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/extract"
	"github.com/appversal/appstorys-cli/internal/output"
	"github.com/appversal/appstorys-cli/internal/project"
)

func newScanCmd() *cobra.Command {
	var (
		format   string
		kinds    []string
		name     string
		platform []string
		groupBy  string
	)

	cmd := &cobra.Command{
		Use:   "scan",
		Short: "List init, screen, event, overlay-host, placement and tag call sites",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			restrict, err := parsePlatforms(platform)
			if err != nil {
				return &CLIError{Code: ExitUsageOrConfig, Err: err}
			}

			sites, _, err := scanProject(app, restrict)
			if err != nil {
				return err
			}
			sites = filterSites(sites, kinds, name)
			sortSites(sites, groupBy)

			switch format {
			case "json":
				return output.WriteJSON(app.Out, sites)
			case "sarif":
				return writeScanSARIF(app.Out, sites)
			case "md":
				return output.WriteMarkdownTable(app.Out, scanHeaders, scanRows(sites))
			case "table", "":
				return output.WriteTable(app.Out, scanHeaders, scanRows(sites))
			default:
				return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("unknown --format %q", format)}
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json|sarif|md")
	cmd.Flags().StringSliceVar(&kinds, "kind", nil, "filter to these call-site kinds (init,screen,event,overlay-host,placement,tag)")
	cmd.Flags().StringVar(&name, "name", "", "filter to call sites whose name contains this substring")
	cmd.Flags().StringSliceVar(&platform, "platform", nil, "restrict detection to these platforms")
	cmd.Flags().StringVar(&groupBy, "group-by", "file", "sort order: name|file")
	return cmd
}

var scanHeaders = []string{"kind", "name", "file:line", "detail"}

func scanRows(sites []extract.CallSite) [][]string {
	rows := make([][]string, 0, len(sites))
	for _, s := range sites {
		rows = append(rows, []string{s.Kind, scanDisplayName(s), fmt.Sprintf("%s:%d", s.File, s.Line), scanDetail(s)})
	}
	return rows
}

func scanDisplayName(s extract.CallSite) string {
	switch {
	case s.Dynamic:
		return "<dynamic>"
	case s.Name == "":
		return "-"
	default:
		return s.Name
	}
}

func scanDetail(s extract.CallSite) string {
	switch s.Kind {
	case "screen":
		if s.Position != "" {
			return "positions: " + s.Position
		}
	case "event":
		if len(s.Properties) > 0 {
			keys := make([]string, 0, len(s.Properties))
			for k := range s.Properties {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts := make([]string, 0, len(keys))
			for _, k := range keys {
				parts = append(parts, fmt.Sprintf("%s:%s", k, s.Properties[k]))
			}
			return "props: " + strings.Join(parts, ",")
		}
	case "placement":
		if s.Position != "" {
			return "position: " + s.Position
		}
	}
	return "-"
}

func writeScanSARIF(w io.Writer, sites []extract.CallSite) error {
	results := make([]output.SARIFResult, 0, len(sites))
	for _, s := range sites {
		results = append(results, output.SARIFResult{
			RuleID:  "scan:" + s.Kind,
			Level:   "note",
			Message: fmt.Sprintf("%s %s via %s", s.Kind, scanDisplayName(s), s.Via),
			File:    s.File,
			Line:    s.Line,
			Col:     s.Col,
		})
	}
	return output.WriteSARIF(w, "appstorys-cli", Version, results)
}

func filterSites(sites []extract.CallSite, kinds []string, name string) []extract.CallSite {
	if len(kinds) == 0 && name == "" {
		return sites
	}
	kindSet := make(map[string]bool, len(kinds))
	for _, k := range kinds {
		kindSet[k] = true
	}
	out := sites[:0:0]
	for _, s := range sites {
		if len(kindSet) > 0 && !kindSet[s.Kind] {
			continue
		}
		if name != "" && !strings.Contains(s.Name, name) {
			continue
		}
		out = append(out, s)
	}
	return out
}

func sortSites(sites []extract.CallSite, groupBy string) {
	sort.SliceStable(sites, func(i, j int) bool {
		if groupBy == "name" {
			if sites[i].Name != sites[j].Name {
				return sites[i].Name < sites[j].Name
			}
			return sites[i].File < sites[j].File
		}
		if sites[i].File != sites[j].File {
			return sites[i].File < sites[j].File
		}
		return sites[i].Line < sites[j].Line
	})
}

func parsePlatforms(values []string) ([]project.Platform, error) {
	if len(values) == 0 {
		return nil, nil
	}
	out := make([]project.Platform, 0, len(values))
	for _, v := range values {
		p := project.Platform(v)
		if _, ok := project.ByPlatform(p); !ok {
			return nil, fmt.Errorf("unknown platform %q", v)
		}
		out = append(out, p)
	}
	return out, nil
}
