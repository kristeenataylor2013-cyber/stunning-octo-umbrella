// Package taskrunner implements octo's "run" subcommand: loading a simple
// YAML file of named shell-command tasks and executing one of them.
package taskrunner

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"

	"gopkg.in/yaml.v3"
)

// Config is the shape of octo's task configuration file, e.g.:
//
//	tasks:
//	  build: go build ./...
//	  test: go test ./...
type Config struct {
	Tasks map[string]string `yaml:"tasks"`
}

// ParseConfig parses raw YAML bytes into a Config. It is a pure function so
// it can be tested without touching the filesystem.
func ParseConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse task config: %w", err)
	}
	if cfg.Tasks == nil {
		cfg.Tasks = map[string]string{}
	}
	return &cfg, nil
}

// LoadConfig reads and parses the task config file at path.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseConfig(data)
}

// TaskNames returns the configured task names in alphabetical order.
func (c *Config) TaskNames() []string {
	names := make([]string, 0, len(c.Tasks))
	for name := range c.Tasks {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Lookup returns the shell command configured for the named task.
func (c *Config) Lookup(name string) (string, bool) {
	cmdStr, ok := c.Tasks[name]
	return cmdStr, ok
}

// Executor runs a shell command, streaming its output to stdout/stderr, and
// reports the command's exit code. It is an interface so RunTask's control
// flow can be unit tested without actually spawning a shell.
type Executor interface {
	Run(command string, stdout, stderr io.Writer) (exitCode int, err error)
}

// ShellExecutor is an Executor backed by `sh -c <command>`.
type ShellExecutor struct{}

// Run implements Executor.
func (ShellExecutor) Run(command string, stdout, stderr io.Writer) (int, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	cmd.Stdin = os.Stdin
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return exitErr.ExitCode(), nil
	}
	return -1, err
}

// RunTask looks up name in cfg and runs it via exec, streaming output to
// stdout/stderr. It returns the task's exit code (0 on success) and an
// error only when the task could not be found or could not be started at
// all (as opposed to simply exiting non-zero).
func RunTask(cfg *Config, name string, exec Executor, stdout, stderr io.Writer) (int, error) {
	cmdStr, ok := cfg.Lookup(name)
	if !ok {
		return -1, fmt.Errorf("task %q not found in config (available: %v)", name, cfg.TaskNames())
	}
	return exec.Run(cmdStr, stdout, stderr)
}
