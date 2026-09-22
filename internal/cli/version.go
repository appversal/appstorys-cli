package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/symbols"
)

// Version is the CLI's own version, overridable at build time via
// -ldflags "-X .../internal/cli.Version=...".
var Version = "dev"

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the CLI version and the supported SDK version per platform",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			maps, err := symbols.Load()
			if err != nil {
				return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("loading SDK symbol maps: %w", err)}
			}

			fmt.Fprintf(app.Out, "appstorys-cli %s\n", Version)
			fmt.Fprintln(app.Out, "supported SDK versions:")

			platforms := make([]string, 0, len(maps))
			for p := range maps {
				platforms = append(platforms, p)
			}
			sort.Strings(platforms)
			for _, p := range platforms {
				fmt.Fprintf(app.Out, "  %-13s %s\n", p, maps[p].SDKVersion)
			}
			return nil
		},
	}
}
