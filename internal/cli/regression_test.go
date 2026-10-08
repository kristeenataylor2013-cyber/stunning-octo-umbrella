package cli

import (
	"bytes"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/internal/gitutil"
)

func testRepo(t *testing.T) gitutil.ExecRunner {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git unavailable")
	}
	r := gitutil.ExecRunner{Dir: t.TempDir()}
	for _, args := range [][]string{
		{"init", "--initial-branch=main"},
		{"config", "user.name", "Octo Test"},
		{"config", "user.email", "octo@example.com"},
		{"commit", "--allow-empty", "-m", "feat: initial commit"},
	} {
		if _, err := r.Run(args...); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

func TestChangelogRealHistory(t *testing.T) {
	r := testRepo(t)
	cmd, out, _ := newCmdWithBuffers()
	if err := runChangelog(cmd, r, "", "HEAD", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "initial commit") {
		t.Fatalf("root missing: %s", out.String())
	}
	if _, err := r.Run("tag", "v1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.Run("commit", "--allow-empty", "-m", "fix: later commit"); err != nil {
		t.Fatal(err)
	}
	for _, from := range []string{"", "v1"} {
		out.Reset()
		if err := runChangelog(cmd, r, from, "HEAD", ""); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out.String(), "initial commit") || !strings.Contains(out.String(), "later commit") {
			t.Fatalf("incorrect exclusive tag range: %s", out.String())
		}
	}
}

func TestCleanFlagSafety(t *testing.T) {
	r := testRepo(t)
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(r.Dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(oldDir); err != nil {
			t.Error(err)
		}
	})
	for _, tc := range []struct {
		args    []string
		deleted bool
	}{
		{nil, false},
		{[]string{"--dry-run=false"}, false},
		{[]string{"--force=false", "--dry-run=false"}, false},
		{[]string{"--force", "--dry-run=true"}, false},
		{[]string{"--force"}, true},
		{[]string{"--force", "--dry-run=false"}, true},
	} {
		t.Run(strings.Join(tc.args, " "), func(t *testing.T) {
			if _, err := r.Run("branch", "feature/done"); err != nil {
				t.Fatal(err)
			}
			cmd := newCleanCmd()
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			cmd.SetArgs(tc.args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			_, err := r.Run("rev-parse", "--verify", "refs/heads/feature/done")
			if (err != nil) != tc.deleted {
				t.Fatalf("deleted = %v, want %v", err != nil, tc.deleted)
			}
			if !tc.deleted {
				if _, err := r.Run("branch", "-d", "feature/done"); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

func TestChangelogDefaultsUseEndingRef(t *testing.T) {
	r := testRepo(t)
	for _, args := range [][]string{
		{"tag", "v1"},
		{"commit", "--allow-empty", "-m", "fix: older release"},
		{"branch", "older"},
		{"commit", "--allow-empty", "-m", "feat: newer release"},
		{"tag", "v2"},
	} {
		if _, err := r.Run(args...); err != nil {
			t.Fatal(err)
		}
	}
	cmd, out, _ := newCmdWithBuffers()
	if err := runChangelog(cmd, r, "", "older", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "older release") || strings.Contains(out.String(), "newer release") || strings.Contains(out.String(), "initial commit") {
		t.Fatalf("incorrect ending-ref range: %s", out.String())
	}
	if _, err := r.Run("tag", "-d", "v1"); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := runChangelog(cmd, r, "", "older", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "initial commit") || !strings.Contains(out.String(), "older release") || strings.Contains(out.String(), "newer release") {
		t.Fatalf("incorrect untagged ending-ref history: %s", out.String())
	}
}

// advancingRunner deterministically advances a branch between the merged query
// and deletion, including the last moment before the atomic update-ref.
type advancingRunner struct {
	gitutil.ExecRunner
	trigger  string
	name     string
	tip      string
	advanced bool
}

func (r *advancingRunner) Run(args ...string) (string, error) {
	if !r.advanced && args[0] == r.trigger {
		r.advanced = true
		if _, err := r.ExecRunner.Run("update-ref", "refs/heads/"+r.name, r.tip); err != nil {
			return "", err
		}
	}
	return r.ExecRunner.Run(args...)
}

func TestCleanPreservesAdvancedBranch(t *testing.T) {
	for _, trigger := range []string{"update-ref"} {
		t.Run(trigger, func(t *testing.T) {
			r := testRepo(t)
			for _, args := range [][]string{
				{"branch", "victim"},
				{"checkout", "-b", "unmerged"},
				{"commit", "--allow-empty", "-m", "feat: unmerged work"},
				{"checkout", "main"},
			} {
				if _, err := r.Run(args...); err != nil {
					t.Fatal(err)
				}
			}
			tip, err := r.Run("rev-parse", "refs/heads/unmerged")
			if err != nil {
				t.Fatal(err)
			}
			advancing := &advancingRunner{ExecRunner: r, trigger: trigger, name: "victim", tip: strings.TrimSpace(tip)}
			cmd, _, _ := newCmdWithBuffers()
			if err := runClean(cmd, advancing, false, true); err == nil {
				t.Fatal("expected changed-tip deletion error")
			}
			got, err := r.Run("rev-parse", "refs/heads/victim")
			if err != nil || got != tip {
				t.Fatalf("advanced branch lost: %q, %v", got, err)
			}
		})
	}
}

func TestCleanDashBranchAndWorktree(t *testing.T) {
	r := testRepo(t)
	for _, args := range [][]string{
		{"update-ref", "refs/heads/-victim", "HEAD"},
		{"branch", "occupied"},
		{"worktree", "add", t.TempDir(), "occupied"},
	} {
		if _, err := r.Run(args...); err != nil {
			t.Fatal(err)
		}
	}
	cmd, out, _ := newCmdWithBuffers()
	if err := runClean(cmd, r, false, true); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Deleted -victim") {
		t.Fatalf("dash branch not deleted: %s", out.String())
	}
	if _, err := r.Run("rev-parse", "--verify", "refs/heads/-victim"); err == nil {
		t.Fatal("dash branch survived")
	}
	if _, err := r.Run("rev-parse", "--verify", "refs/heads/occupied"); err != nil {
		t.Fatal("worktree branch deleted")
	}
}

func TestCleanPreservesNewlyOccupiedBranch(t *testing.T) {
	r := testRepo(t)
	if _, err := r.Run("branch", "occupied"); err != nil {
		t.Fatal(err)
	}
	candidates, err := gitutil.ListMergedBranchTips(r, "refs/heads/main")
	if err != nil || len(candidates) != 1 {
		t.Fatalf("candidates: %v, %v", candidates, err)
	}
	if _, err := r.Run("worktree", "add", t.TempDir(), "occupied"); err != nil {
		t.Fatal(err)
	}
	deleted, errs := gitutil.DeleteMergedBranches(r, candidates)
	if len(deleted) != 0 || len(errs) != 1 {
		t.Fatalf("occupied branch: deleted %v, errors %v", deleted, errs)
	}
	if _, err := r.Run("rev-parse", "--verify", "refs/heads/occupied"); err != nil {
		t.Fatal("worktree branch deleted")
	}
}
