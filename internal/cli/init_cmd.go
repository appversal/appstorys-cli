package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/appversal/appstorys-cli/internal/patch"
	"github.com/appversal/appstorys-cli/internal/project"
	"github.com/appversal/appstorys-cli/internal/suggest"
)

func newInitCmd() *cobra.Command {
	var apply bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Add the AppStorys dependency and init call for the detected platform",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			app := FromContext(cmd.Context())

			supported, err := detectSupported(app, nil)
			if err != nil {
				return err
			}

			var suggestions []suggest.Suggestion
			var manualNotes []string
			for _, info := range supported {
				switch info.Platform {
				case project.Android:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					classes, err := allKotlinClasses(app)
					if err != nil {
						return err
					}
					s, err := planInitAndroid(app, sites, classes)
					if err != nil {
						return err
					}
					if len(s) > 0 {
						manualNotes = append(manualNotes, "Android: confirm a navigateToScreen lambda is set (an unset one crashes on first deep-link click), and check the Gradle edits landed correctly if any repositories/dependencies/android block wasn't found automatically.")
					}
					suggestions = append(suggestions, s...)
				case project.ReactNative:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					s, err := planInitReactNative(app, sites)
					if err != nil {
						return err
					}
					if len(s) > 0 {
						manualNotes = append(manualNotes, "React Native: wire the npm dependency + its 14 peer dependencies, the Babel plugin, GestureHandlerRootView + SafeAreaProvider at root, and appstorys.d.ts in tsconfig.json — none of that is automated yet.")
					}
					suggestions = append(suggestions, s...)
				case project.Flutter:
					sites, err := scanPlatform(app, info)
					if err != nil {
						return err
					}
					s, err := planInitFlutter(app, sites)
					if err != nil {
						return err
					}
					if len(s) > 0 {
						manualNotes = append(manualNotes, "Flutter: add appstorys_sdk_3_0 to pubspec.yaml and enable Android core library desugaring — not yet automated.")
					}
					suggestions = append(suggestions, s...)
				default:
					return &CLIError{Code: ExitProjectUnsupported, Err: fmt.Errorf(
						"init doesn't support %s yet (only android, react-native and flutter are implemented so far)", info.Platform)}
				}
			}

			if len(suggestions) == 0 {
				fmt.Fprintln(app.Out, "An AppStorys init call already exists; nothing to add. Run `doctor` to check it's in the right place.")
				return nil
			}

			if !apply {
				for _, s := range suggestions {
					original, exists := readTarget(app.Root, s.File)
					fmt.Fprintf(app.Out, "# %s\n", s.Reason)
					fmt.Fprint(app.Out, patch.Render(toEdit(s, !exists), original))
					fmt.Fprintln(app.Out)
				}
				return nil
			}

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
			fmt.Fprintf(app.Out, "Applied %d change(s).\n", len(suggestions))
			for _, note := range manualNotes {
				fmt.Fprintln(app.Out, "Still needed manually —", note)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&apply, "apply", false, "write the changes (requires a clean git tree)")
	return cmd
}
