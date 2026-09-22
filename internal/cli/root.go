package cli

import (
	"errors"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/config"
)

// NewRootCmd builds the appstorys-cli root command with its global flags
// and subcommands.
func NewRootCmd() *cobra.Command {
	var (
		root    string
		cfgPath string
		noColor bool
		quiet   bool
		verbose bool
	)

	root = "."

	cmd := &cobra.Command{
		Use:           "appstorys-cli",
		Short:         "Inventory, validate, register and suggest AppStorys SDK instrumentation",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(config.LoadOptions{
				Root:         root,
				ExplicitPath: cfgPath,
				Env:          environMap(),
			})
			if err != nil {
				return &CLIError{Code: ExitUsageOrConfig, Err: err}
			}
			app := &App{
				Config:  cfg,
				Root:    root,
				Out:     cmd.OutOrStdout(),
				ErrOut:  cmd.ErrOrStderr(),
				NoColor: noColor,
				Quiet:   quiet,
				Verbose: verbose,
			}
			cmd.SetContext(withApp(cmd.Context(), app))
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&root, "root", root, "project root to operate on")
	cmd.PersistentFlags().StringVar(&cfgPath, "config", "", "path to .appstorys-cli.yaml (default: <root>/.appstorys-cli.yaml)")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	cmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "suppress non-essential output")
	cmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "print additional diagnostic detail")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newScanCmd())
	cmd.AddCommand(newLintCmd())
	cmd.AddCommand(newValidateCmd())
	cmd.AddCommand(newEventsCmd())
	cmd.AddCommand(newDoctorCmd())

	return cmd
}

// Execute runs the root command and returns the process exit code.
func Execute() int {
	err := NewRootCmd().Execute()
	if err == nil {
		return ExitOK
	}
	var cliErr *CLIError
	if errors.As(err, &cliErr) {
		return cliErr.Code
	}
	return ExitUsageOrConfig
}

func environMap() map[string]string {
	out := make(map[string]string)
	for _, kv := range os.Environ() {
		if !strings.HasPrefix(kv, "APPSTORYS_CLI_") {
			continue
		}
		parts := strings.SplitN(kv, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}
