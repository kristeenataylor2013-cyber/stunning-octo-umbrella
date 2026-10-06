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
