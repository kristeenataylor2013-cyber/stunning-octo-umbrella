// Package cli wires up octo's cobra commands. Business logic lives in the
// internal/gitutil, internal/changelog, and internal/taskrunner packages so
// it can be unit tested independently of CLI/flag parsing.
package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/internal/version"
)

// exitError lets a subcommand propagate a specific process exit code (for
// example, the exit code of a shell command run by `octo run`) through
// cobra's error-returning RunE without octo printing a spurious "Error:"
// message for what is really just a pass-through exit status.
type exitError struct{ code int }

func (e *exitError) Error() string { return fmt.Sprintf("exit status %d", e.code) }

// newExitError builds an error that Execute will translate into the given
// process exit code without printing anything extra.
func newExitError(code int) error { return &exitError{code: code} }

// NewRootCmd builds octo's root command and registers all subcommands.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "octo",
		Short:        "octo is a toolbox of everyday git and project maintenance commands",
		Long:         `octo bundles small, focused repository maintenance utilities.`,
		Version:      version.Version,
		SilenceUsage: true,
		Example: `  octo clean --dry-run
  octo changelog --from v1.0.0 --to HEAD
  octo run build`,
	}

	root.AddCommand(newCleanCmd())
	root.AddCommand(newChangelogCmd())
	root.AddCommand(newRunCmd())

	return root
}

// Execute runs octo's root command and returns the process exit code.
func Execute() int {
	root := NewRootCmd()
	root.SilenceErrors = true

	err := root.Execute()
	if err == nil {
		return 0
	}

	var ee *exitError
	if errors.As(err, &ee) {
		return ee.code
	}

	fmt.Fprintln(os.Stderr, "Error:", err)
	return 1
}
