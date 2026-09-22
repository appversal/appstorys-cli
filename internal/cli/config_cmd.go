package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage .appstorys-cli.yaml",
	}
	cmd.AddCommand(newConfigInitCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Write a default .appstorys-cli.yaml in the project root",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())
			path := filepath.Join(app.Root, ".appstorys-cli.yaml")

			if !force {
				if _, err := os.Stat(path); err == nil {
					return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("%s already exists (use --force to overwrite)", path)}
				}
			}

			if err := os.WriteFile(path, config.Template(), 0o644); err != nil {
				return &CLIError{Code: ExitUsageOrConfig, Err: fmt.Errorf("writing %s: %w", path, err)}
			}

			fmt.Fprintf(app.Out, "wrote %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing .appstorys-cli.yaml")
	return cmd
}
