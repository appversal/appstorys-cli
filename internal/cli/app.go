package cli

import (
	"context"
	"io"

	"github.com/appversal/appstorys-cli/internal/config"
)

// App holds everything a subcommand needs: the resolved config, the
// project root it's operating on, output streams, and global display
// flags. It's built once in the root command's PersistentPreRunE and
// threaded through the command context.
type App struct {
	Config  *config.Config
	Root    string
	Out     io.Writer
	ErrOut  io.Writer
	NoColor bool
	Quiet   bool
	Verbose bool
}

type appContextKey struct{}

func withApp(ctx context.Context, app *App) context.Context {
	return context.WithValue(ctx, appContextKey{}, app)
}

// FromContext returns the App attached to ctx by the root command. It
// panics if called outside a command run, which would be a programming
// error (every subcommand goes through PersistentPreRunE first).
func FromContext(ctx context.Context) *App {
	app, ok := ctx.Value(appContextKey{}).(*App)
	if !ok {
		panic("cli: no App in context; called outside a command run")
	}
	return app
}
