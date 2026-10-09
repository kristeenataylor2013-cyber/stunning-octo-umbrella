package gitutil

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRunner is an in-memory Runner used to unit test the pure
// decision/parsing logic without invoking a real git binary.
type fakeRunner struct {
	outputs map[string]string
	errs    map[string]error
	calls   [][]string
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{outputs: map[string]string{}, errs: map[string]error{}}
}

func (f *fakeRunner) key(args []string) string { return strings.Join(args, " ") }

func (f *fakeRunner) Run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	key := f.key(args)
	return f.outputs[key], f.errs[key]
}

func TestParseMergedBranches(t *testing.T) {
	tests := []struct {
		name          string
		output        string
		defaultBranch string
		want          []string
	}{
		{
			name:          "excludes default branch and blank lines",
			output:        "main\nfeature/old\n\nrelease/1.0\n",
			defaultBranch: "main",
			want:          []string{"feature/old", "release/1.0"},
		},
		{
			name:          "strips current branch marker",
			output:        "* main\n  feature/x\n",
			defaultBranch: "main",
			want:          []string{"feature/x"},
		},
		{
			name:          "always excludes main and master even if not the default",
			output:        "develop\nmain\nmaster\nfeature/y\n",
			defaultBranch: "develop",
			want:          []string{"feature/y"},
		},
		{
			name:          "empty output yields nil",
			output:        "",
			defaultBranch: "main",
			want:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseMergedBranches(tt.output, tt.defaultBranch)
			if !equalSlices(got, tt.want) {
				t.Fatalf("ParseMergedBranches(%q, %q) = %v, want %v", tt.output, tt.defaultBranch, got, tt.want)
			}
		})
	}
}

func TestDetectDefaultBranch_PrefersOriginHEAD(t *testing.T) {
	r := newFakeRunner()
	r.outputs["symbolic-ref refs/remotes/origin/HEAD"] = "refs/remotes/origin/develop\n"

	got, err := DetectDefaultBranch(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "refs/remotes/origin/develop" {
		t.Fatalf("got %q, want %q", got, "refs/remotes/origin/develop")
	}
}

func TestDetectDefaultBranch_FallsBackToMain(t *testing.T) {
	r := newFakeRunner()
	r.errs["symbolic-ref refs/remotes/origin/HEAD"] = errors.New("no such ref")
	r.errs["rev-parse --verify --quiet refs/heads/master"] = errors.New("not found")
	// "main" verify succeeds (no error, empty output is fine).

	got, err := DetectDefaultBranch(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "refs/heads/main" {
		t.Fatalf("got %q, want %q", got, "refs/heads/main")
	}
}

func TestDetectDefaultBranch_FallsBackToMaster(t *testing.T) {
	r := newFakeRunner()
	r.errs["symbolic-ref refs/remotes/origin/HEAD"] = errors.New("no such ref")
	r.errs["rev-parse --verify --quiet refs/heads/main"] = errors.New("not found")

	got, err := DetectDefaultBranch(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "refs/heads/master" {
		t.Fatalf("got %q, want %q", got, "refs/heads/master")
	}
}

func TestDetectDefaultBranch_NoneFound(t *testing.T) {
	r := newFakeRunner()
	r.errs["symbolic-ref refs/remotes/origin/HEAD"] = errors.New("no such ref")
	r.errs["rev-parse --verify --quiet refs/heads/main"] = errors.New("not found")
	r.errs["rev-parse --verify --quiet refs/heads/master"] = errors.New("not found")

	_, err := DetectDefaultBranch(r)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestListMergedBranches(t *testing.T) {
	r := newFakeRunner()
	r.outputs["branch --format=%(refname:lstrip=2) --merged main"] = "main\nfeature/a\nfeature/b\n"

	got, err := ListMergedBranches(r, "main")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []string{"feature/a", "feature/b"}
	if !equalSlices(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestDeleteBranches(t *testing.T) {
	r := newFakeRunner()
	r.errs["branch -d -- bad-branch"] = errors.New("not fully merged")

	deleted, errs := DeleteBranches(r, []string{"good-branch", "bad-branch"}, false)

	if !equalSlices(deleted, []string{"good-branch"}) {
		t.Fatalf("deleted = %v, want [good-branch]", deleted)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}

	// Verify the delete flag used.
	for _, call := range r.calls {
		if len(call) >= 1 && call[0] == "branch" {
			if call[1] != "-d" {
				t.Fatalf("expected -d flag for non-force delete, got %v", call)
			}
		}
	}
}

func TestDeleteBranches_Force(t *testing.T) {
	r := newFakeRunner()
	_, _ = DeleteBranches(r, []string{"b1"}, true)
	found := false
	for _, call := range r.calls {
		if len(call) == 4 && call[0] == "branch" && call[1] == "-D" && call[2] == "--" && call[3] == "b1" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a `branch -D b1` call, got %v", r.calls)
	}
}

func TestLatestTag(t *testing.T) {
	r := newFakeRunner()
	r.outputs["describe --tags --abbrev=0 HEAD"] = "v1.2.3\n"
	got, err := LatestTag(r, "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v1.2.3" {
		t.Fatalf("got %q, want %q", got, "v1.2.3")
	}
}

func TestFirstCommit(t *testing.T) {
	r := newFakeRunner()
	r.outputs["rev-list --max-parents=0 HEAD"] = "abc123\n"
	got, err := FirstCommit(r, "HEAD")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "abc123" {
		t.Fatalf("got %q, want %q", got, "abc123")
	}
}

func TestFirstCommit_NoCommits(t *testing.T) {
	r := newFakeRunner()
	r.outputs["rev-list --max-parents=0 HEAD"] = ""
	_, err := FirstCommit(r, "HEAD")
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// TestExecRunner_Integration exercises ExecRunner (and therefore
// ListMergedBranches/DetectDefaultBranch) against a real throwaway git
// repository, skipping if git isn't available in the environment.
func TestExecRunner_Integration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed, skipping integration test")
	}

	dir := t.TempDir()
	runner := ExecRunner{Dir: dir}

	run := func(args ...string) {
		if _, err := runner.Run(args...); err != nil {
			t.Fatalf("git %v: %v", args, err)
		}
	}

	run("init", "--initial-branch=main")
	run("config", "user.email", "octo@example.com")
	run("config", "user.name", "Octo Test")
	writeFile(t, dir, "README.md", "hello")
	run("add", ".")
	run("commit", "-m", "chore: initial commit")

	run("checkout", "-b", "feature/done")
	writeFile(t, dir, "feature.txt", "feature")
	run("add", ".")
	run("commit", "-m", "feat: add feature")
	run("checkout", "main")
	run("merge", "--no-ff", "-m", "merge feature", "feature/done")

	run("checkout", "-b", "feature/unmerged")
	writeFile(t, dir, "wip.txt", "wip")
	run("add", ".")
	run("commit", "-m", "feat: work in progress")
	run("checkout", "main")

	branches, err := ListMergedBranches(runner, "main")
	if err != nil {
		t.Fatalf("ListMergedBranches: %v", err)
	}
	if !equalSlices(branches, []string{"feature/done"}) {
		t.Fatalf("got %v, want [feature/done]", branches)
	}

	deleted, errs := DeleteBranches(runner, branches, false)
	if len(errs) != 0 {
		t.Fatalf("unexpected delete errors: %v", errs)
	}
	if !equalSlices(deleted, []string{"feature/done"}) {
		t.Fatalf("deleted = %v, want [feature/done]", deleted)
	}

	// A same-named tag must not replace the local default branch.
	run("tag", "main", "HEAD~1")
	ref, err := DetectDefaultBranch(runner)
	if err != nil || ref != "refs/heads/main" {
		t.Fatalf("default with colliding tag = %q, %v", ref, err)
	}
	run("branch", "feature/done", "HEAD")
	branches, err = ListMergedBranches(runner, ref)
	if err != nil || !equalSlices(branches, []string{"feature/done"}) {
		t.Fatalf("merged branches with colliding tag = %v, %v", branches, err)
	}

	// origin/HEAD remains usable even without its corresponding local branch.
	run("update-ref", "refs/remotes/origin/develop", "refs/heads/main")
	run("symbolic-ref", "refs/remotes/origin/HEAD", "refs/remotes/origin/develop")
	run("checkout", "feature/unmerged")
	run("branch", "-D", "main")
	ref, err = DetectDefaultBranch(runner)
	if err != nil || ref != "refs/remotes/origin/develop" {
		t.Fatalf("remote default = %q, %v", ref, err)
	}
	branches, err = ListMergedBranches(runner, ref)
	if err != nil || !equalSlices(branches, []string{"feature/done"}) {
		t.Fatalf("remote merged branches = %v, %v", branches, err)
	}
	run("branch", "develop", "refs/remotes/origin/develop")
	branches, err = ListMergedBranches(runner, ref)
	if err != nil || !equalSlices(branches, []string{"feature/done"}) {
		t.Fatalf("default branch must be protected: %v, %v", branches, err)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
