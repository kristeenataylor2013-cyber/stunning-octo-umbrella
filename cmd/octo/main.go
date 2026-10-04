// Command octo is a small toolbox of everyday git and project maintenance
// utilities. See internal/cli for the subcommand implementations.
package main

import (
	"os"

	"github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
