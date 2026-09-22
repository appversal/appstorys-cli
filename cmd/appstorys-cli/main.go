// Command appstorys-cli inventories, validates, registers and suggests
// AppStorys SDK instrumentation in customer apps.
package main

import (
	"os"

	"github.com/appversal/appstorys-cli/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
