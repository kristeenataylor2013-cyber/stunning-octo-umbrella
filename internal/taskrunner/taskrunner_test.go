package taskrunner

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseConfig(t *testing.T) {
	data := []byte("tasks:\n  build: go build ./...\n  test: go test ./...\n")
	cfg, err := ParseConfig(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := cfg.Lookup("build"); !ok || got != "go build ./..." {
		t.Fatalf("Lookup(build) = %q, %v", got, ok)
	}
	if got, ok := cfg.Lookup("test"); !ok || got != "go test ./..." {
		t.Fatalf("Lookup(test) = %q, %v", got, ok)
	}
	if _, ok := cfg.Lookup("missing"); ok {
		t.Fatal("expected Lookup(missing) to report not found")
	}
}

func TestParseConfig_EmptyTasks(t *testing.T) {
	cfg, err := ParseConfig([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Tasks) != 0 {
		t.Fatalf("expected no tasks, got %v", cfg.Tasks)
	}
}

func TestParseConfig_InvalidYAML(t *testing.T) {
	_, err := ParseConfig([]byte("tasks: [this is not a map"))
	if err == nil {
		t.Fatal("expected an error for invalid YAML")
	}
}

func TestConfig_TaskNames(t *testing.T) {
	cfg := &Config{Tasks: map[string]string{"zeta": "echo z", "alpha": "echo a", "mid": "echo m"}}
	got := cfg.TaskNames()
	want := []string{"alpha", "mid", "zeta"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "octo.yaml")
	if err := os.WriteFile(path, []byte("tasks:\n  hello: echo hi\n"), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got, ok := cfg.Lookup("hello"); !ok || got != "echo hi" {
		t.Fatalf("Lookup(hello) = %q, %v", got, ok)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "does-not-exist.yaml"))
	if err == nil {
		t.Fatal("expected an error for a missing config file")
	}
}

// executorFunc adapts a plain function to the Executor interface so tests
// can script RunTask's interaction with it without spawning a real shell.
type executorFunc func(command string, stdout, stderr io.Writer) (int, error)

func (f executorFunc) Run(command string, stdout, stderr io.Writer) (int, error) {
	return f(command, stdout, stderr)
}

func TestRunTask_NotFound(t *testing.T) {
	cfg := &Config{Tasks: map[string]string{}}
	_, err := RunTask(cfg, "missing", ShellExecutor{}, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected an error for a missing task")
	}
}

func TestRunTask_DelegatesToExecutor(t *testing.T) {
	cfg := &Config{Tasks: map[string]string{"greet": "echo hello"}}

	var called string
	exec := executorFunc(func(command string, stdout, stderr io.Writer) (int, error) {
		called = command
		return 3, nil
	})

	code, err := RunTask(cfg, "greet", exec, &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 3 {
		t.Fatalf("code = %d, want 3", code)
	}
	if called != "echo hello" {
		t.Fatalf("executor was called with %q, want %q", called, "echo hello")
	}
}

func TestRunTask_ExecutorError(t *testing.T) {
	cfg := &Config{Tasks: map[string]string{"greet": "echo hello"}}
	exec := executorFunc(func(command string, stdout, stderr io.Writer) (int, error) {
		return -1, errors.New("could not start process")
	})

	_, err := RunTask(cfg, "greet", exec, &bytes.Buffer{}, &bytes.Buffer{})
	if err == nil {
		t.Fatal("expected error to propagate from the executor")
	}
}

func TestShellExecutor_Run(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := ShellExecutor{}.Run("echo hello-from-octo", &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if stdout.String() != "hello-from-octo\n" {
		t.Fatalf("stdout = %q, want %q", stdout.String(), "hello-from-octo\n")
	}
}

func TestShellExecutor_NonZeroExit(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code, err := ShellExecutor{}.Run("exit 7", &stdout, &stderr)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code != 7 {
		t.Fatalf("code = %d, want 7", code)
	}
}

func TestShellExecutor_SignalExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("requires POSIX signals")
	}
	code, err := (ShellExecutor{}).Run("kill -TERM $$", &bytes.Buffer{}, &bytes.Buffer{})
	if err != nil || code != 143 {
		t.Fatalf("signal exit = %d, %v; want 143, nil", code, err)
	}
}
