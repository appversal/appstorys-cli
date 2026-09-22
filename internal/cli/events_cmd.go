package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/catalog"
	"github.com/appversal/appstorys-cli/internal/output"
)

func newEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Work with the deduplicated events/screens/positions catalog",
	}
	cmd.AddCommand(newEventsListCmd())
	return cmd
}

func newEventsListCmd() *cobra.Command {
	var (
		format string
		only   string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Deduplicated catalog of events, screens and widget positions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			switch only {
			case "", "events", "screens", "positions":
			default:
				return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("--only must be events, screens or positions, got %q", only)}
			}

			sites, _, err := scanProject(app, nil)
			if err != nil {
				return err
			}
			c := catalog.Build(sites)

			switch format {
			case "json":
				return output.WriteJSON(app.Out, selectCatalog(c, only))
			case "md":
				return writeCatalogTable(app.Out, c, only, output.WriteMarkdownTable)
			case "table", "":
				return writeCatalogTable(app.Out, c, only, output.WriteTable)
			default:
				return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("unknown --format %q", format)}
			}
		},
	}

	cmd.Flags().StringVar(&format, "format", "table", "output format: table|json|md")
	cmd.Flags().StringVar(&only, "only", "", "restrict to events|screens|positions (default: all)")
	return cmd
}

func selectCatalog(c catalog.Catalog, only string) any {
	switch only {
	case "events":
		return c.Events
	case "screens":
		return c.Screens
	case "positions":
		return c.Positions
	default:
		return c
	}
}

var catalogHeaders = []string{"kind", "name", "platforms", "call-sites", "properties"}

func catalogRows(items []catalog.Item) [][]string {
	rows := make([][]string, 0, len(items))
	for _, it := range items {
		platforms := make([]string, len(it.Platforms))
		for i, p := range it.Platforms {
			platforms[i] = string(p)
		}
		props := "-"
		if len(it.Properties) > 0 {
			keys := make([]string, 0, len(it.Properties))
			for k := range it.Properties {
				keys = append(keys, k)
			}
			propParts := make([]string, 0, len(keys))
			for _, k := range keys {
				propParts = append(propParts, fmt.Sprintf("%s:%s", k, it.Properties[k]))
			}
			props = strings.Join(propParts, ",")
		}
		rows = append(rows, []string{it.Kind, it.Name, strings.Join(platforms, ","), fmt.Sprintf("%d", it.CallSites), props})
	}
	return rows
}

func writeCatalogTable(w io.Writer, c catalog.Catalog, only string, writer func(w io.Writer, headers []string, rows [][]string) error) error {
	var rows [][]string
	switch only {
	case "events":
		rows = catalogRows(c.Events)
	case "screens":
		rows = catalogRows(c.Screens)
	case "positions":
		rows = catalogRows(c.Positions)
	default:
		rows = append(rows, catalogRows(c.Events)...)
		rows = append(rows, catalogRows(c.Screens)...)
		rows = append(rows, catalogRows(c.Positions)...)
	}
	return writer(w, catalogHeaders, rows)
}
