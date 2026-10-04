package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/internal/taskrunner"
)

func newRunCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "run <task>",
		Short: "Run a named task from the octo task config file",
		Long: `run reads a YAML config file (default "octo.yaml" in the current
directory) defining named tasks as shell commands, e.g.:

  tasks:
    build: go build ./...
    test: go test ./...

and runs the named task, streaming its output live and exiting with the
task's exit code.`,
		Example: `  octo run build
  octo run test --config tasks.yaml`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRun(cmd, configPath, args[0])
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "octo.yaml", "path to the task config file")
	return cmd
}

func runRun(cmd *cobra.Command, configPath, task string) error {
	cfg, err := taskrunner.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("load config %s: %w", configPath, err)
	}

	code, err := taskrunner.RunTask(cfg, task, taskrunner.ShellExecutor{}, cmd.OutOrStdout(), cmd.ErrOrStderr())
	if err != nil {
		return err
	}
	if code != 0 {
		return newExitError(code)
	}
	return nil
}
