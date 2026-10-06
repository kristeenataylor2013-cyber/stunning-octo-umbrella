// Package gitutil contains small, dependency-light helpers for interacting
// with a local git repository. The Runner interface decouples the pure
// parsing/decision logic from the actual `git` binary so it can be unit
// tested without a real repository.
package gitutil

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// Runner executes a git subcommand (with the given arguments) and returns
// its combined output.
type Runner interface {
	Run(args ...string) (string, error)
}

// ExecRunner is a Runner backed by the real `git` binary on PATH.
type ExecRunner struct {
	// Dir is the working directory git should run in. Empty means the
	// current process working directory.
	Dir string
}

// Run implements Runner using os/exec.
func (e ExecRunner) Run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if e.Dir != "" {
		cmd.Dir = e.Dir
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	if err != nil {
		return out.String(), fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(out.String()))
	}
	return out.String(), nil
}

// CurrentBranch returns the name of the currently checked out branch.
func CurrentBranch(r Runner) (string, error) {
	out, err := r.Run("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// DetectDefaultBranch tries to determine the repository's default branch.
// It first looks at the remote's symbolic HEAD (origin/HEAD), then falls
// back to checking whether a local "main" or "master" branch exists.
func DetectDefaultBranch(r Runner) (string, error) {
	if out, err := r.Run("symbolic-ref", "refs/remotes/origin/HEAD"); err == nil {
		name := strings.TrimSpace(out)
		if name != "" {
			return name, nil
		}
	}
	for _, candidate := range []string{"refs/heads/main", "refs/heads/master"} {
		if _, err := r.Run("rev-parse", "--verify", "--quiet", candidate); err == nil {
			return candidate, nil
		}
	}
	return "", errors.New("could not detect default branch (tried origin/HEAD, main, master)")
}

// ParseMergedBranches parses the newline-separated output of
// `git branch --format=%(refname:lstrip=2) --merged <defaultBranch>` and
// returns the branch names, excluding the default branch itself and any
// blank lines. It is a pure function so it can be tested without a real
// git repository.
func ParseMergedBranches(output, defaultBranch string) []string {
	skip := map[string]bool{
		strings.TrimPrefix(strings.TrimPrefix(defaultBranch, "refs/heads/"), "refs/remotes/origin/"): true,
		"main":   true,
		"master": true,
	}
	var result []string
	for _, line := range strings.Split(output, "\n") {
		name := strings.TrimSpace(line)
		name = strings.TrimPrefix(name, "* ")
		name = strings.TrimSpace(name)
		if name == "" || skip[name] {
			continue
		}
		result = append(result, name)
	}
	return result
}

// ListMergedBranches runs `git branch --merged` against defaultBranch and
// returns the parsed branch names.
func ListMergedBranches(r Runner, defaultBranch string) ([]string, error) {
	out, err := r.Run("branch", "--format=%(refname:lstrip=2)", "--merged", defaultBranch)
	if err != nil {
		return nil, err
	}
	return ParseMergedBranches(out, defaultBranch), nil
}

// DeleteBranches deletes each of the given local branches using `git branch
// -d` (or `-D` when force is true). It returns the branches that were
// successfully deleted and any per-branch errors encountered.
func DeleteBranches(r Runner, branches []string, force bool) (deleted []string, errs []error) {
	flag := "-d"
	if force {
		flag = "-D"
	}
	for _, b := range branches {
		if _, err := r.Run("branch", flag, b); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", b, err))
			continue
		}
		deleted = append(deleted, b)
	}
	return deleted, errs
}

// LatestTag returns the most recent reachable tag from ref, as reported by
// `git describe --tags --abbrev=0`.
func LatestTag(r Runner, ref string) (string, error) {
	out, err := r.Run("describe", "--tags", "--abbrev=0", ref)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

// FirstCommit returns the hash of the repository's first (root) commit.
func FirstCommit(r Runner, ref string) (string, error) {
	out, err := r.Run("rev-list", "--max-parents=0", ref)
	if err != nil {
		return "", err
	}
	fields := strings.Fields(strings.TrimSpace(out))
	if len(fields) == 0 {
		return "", errors.New("no commits found")
	}
	// rev-list can report multiple roots in unusual histories; use the
	// last one, which corresponds to the oldest commit in the common case
	// of a single root.
	return fields[len(fields)-1], nil
}
