package cli

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// fakeRunner is a minimal gitutil.Runner double for exercising the CLI
// wiring (runClean/runChangelog) without a real git repository.
type fakeRunner struct {
	outputs map[string]string
	errs    map[string]error
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	key := strings.Join(args, " ")
	return f.outputs[key], f.errs[key]
}

func newCmdWithBuffers() (*cobra.Command, *bytes.Buffer, *bytes.Buffer) {
	cmd := &cobra.Command{Use: "test"}
	var out, errOut bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	return cmd, &out, &errOut
}

func TestRunClean_DryRunByDefault(t *testing.T) {
	r := &fakeRunner{
		outputs: map[string]string{
			"symbolic-ref refs/remotes/origin/HEAD":                        "",
			"rev-parse --verify --quiet refs/heads/main":                   "",
			"rev-parse --abbrev-ref HEAD":                                  "main",
			"branch --format=%(refname:lstrip=2) --merged refs/heads/main": "main\nfeature/old\n",
		},
		errs: map[string]error{
			"symbolic-ref refs/remotes/origin/HEAD": errors.New("no remote"),
		},
	}

	cmd, out, _ := newCmdWithBuffers()
	if err := runClean(cmd, r, true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "feature/old") {
		t.Fatalf("expected dry-run output to mention feature/old, got:\n%s", out.String())
	}
	if !strings.Contains(out.String(), "dry-run") {
		t.Fatalf("expected dry-run output to say dry-run, got:\n%s", out.String())
	}
}

func TestRunClean_NothingToClean(t *testing.T) {
	r := &fakeRunner{
		outputs: map[string]string{
			"rev-parse --verify --quiet refs/heads/main":                   "",
			"rev-parse --abbrev-ref HEAD":                                  "main",
			"branch --format=%(refname:lstrip=2) --merged refs/heads/main": "main\n",
		},
		errs: map[string]error{
			"symbolic-ref refs/remotes/origin/HEAD": errors.New("no remote"),
		},
	}

	cmd, out, _ := newCmdWithBuffers()
	if err := runClean(cmd, r, true, false); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "No local branches merged") {
		t.Fatalf("expected a no-op message, got:\n%s", out.String())
	}
}

func TestRunClean_ForceDeletes(t *testing.T) {
	r := &fakeRunner{
		outputs: map[string]string{
			"rev-parse --verify --quiet refs/heads/main":                   "",
			"rev-parse --abbrev-ref HEAD":                                  "main",
			"branch --format=%(refname:lstrip=2) --merged refs/heads/main": "main\nfeature/old\n",
			"branch -D feature/old":                                        "",
		},
		errs: map[string]error{
			"symbolic-ref refs/remotes/origin/HEAD": errors.New("no remote"),
		},
	}

	cmd, out, _ := newCmdWithBuffers()
	if err := runClean(cmd, r, false, true); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "Deleted feature/old") {
		t.Fatalf("expected deletion message, got:\n%s", out.String())
	}
}

func TestRunChangelog_DefaultsToFirstCommit(t *testing.T) {
	r := &fakeRunner{
		outputs: map[string]string{
			"rev-list --max-parents=0 HEAD": "abc123\n",
			"log HEAD --format=%H\x1f%s":    "abc123\x1ffeat: initial commit\n",
		},
		errs: map[string]error{
			"describe --tags --abbrev=0": errors.New("no tags"),
		},
	}

	cmd, out, _ := newCmdWithBuffers()
	if err := runChangelog(cmd, r, "", "", ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out.String(), "## Features") {
		t.Fatalf("expected grouped output, got:\n%s", out.String())
	}
}

func TestNewRootCmd_HasSubcommands(t *testing.T) {
	root := NewRootCmd()
	names := map[string]bool{}
	for _, c := range root.Commands() {
		names[c.Name()] = true
	}
	for _, want := range []string{"clean", "changelog", "run"} {
		if !names[want] {
			t.Fatalf("expected root command to have a %q subcommand, got %v", want, names)
		}
	}
}
