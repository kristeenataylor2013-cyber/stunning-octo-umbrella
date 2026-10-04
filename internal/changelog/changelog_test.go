package changelog

import (
	"strings"
	"testing"
)

func TestParseCommit(t *testing.T) {
	tests := []struct {
		name    string
		subject string
		want    ParsedCommit
	}{
		{
			name:    "type and scope",
			subject: "feat(api): add new endpoint",
			want:    ParsedCommit{Type: "feat", Scope: "api", Description: "add new endpoint"},
		},
		{
			name:    "type only",
			subject: "fix: handle nil pointer",
			want:    ParsedCommit{Type: "fix", Description: "handle nil pointer"},
		},
		{
			name:    "breaking change marker",
			subject: "feat!: drop deprecated flag",
			want:    ParsedCommit{Type: "feat", Breaking: true, Description: "drop deprecated flag"},
		},
		{
			name:    "unrecognized type falls back to flat description",
			subject: "bananas: not a real type",
			want:    ParsedCommit{Description: "bananas: not a real type"},
		},
		{
			name:    "non-conventional subject",
			subject: "quick fix for the build",
			want:    ParsedCommit{Description: "quick fix for the build"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Commit{Hash: "deadbeef", Subject: tt.subject}
			got := ParseCommit(c)
			if got.Type != tt.want.Type || got.Scope != tt.want.Scope ||
				got.Breaking != tt.want.Breaking || got.Description != tt.want.Description {
				t.Fatalf("ParseCommit(%q) = %+v, want type=%q scope=%q breaking=%v desc=%q",
					tt.subject, got, tt.want.Type, tt.want.Scope, tt.want.Breaking, tt.want.Description)
			}
		})
	}
}

func TestGroup_TypedCommits(t *testing.T) {
	commits := []Commit{
		{Hash: "1", Subject: "feat: add thing"},
		{Hash: "2", Subject: "fix: fix thing"},
		{Hash: "3", Subject: "feat(cli): another feature"},
		{Hash: "4", Subject: "random commit without a prefix"},
	}

	groups, order, flat := Group(commits)
	if flat {
		t.Fatal("expected flat=false when at least one commit has a recognized type")
	}

	wantOrder := []string{"feat", "fix", "other"}
	if strings.Join(order, ",") != strings.Join(wantOrder, ",") {
		t.Fatalf("order = %v, want %v", order, wantOrder)
	}

	if len(groups["feat"]) != 2 {
		t.Fatalf("expected 2 feat commits, got %d", len(groups["feat"]))
	}
	if len(groups["fix"]) != 1 {
		t.Fatalf("expected 1 fix commit, got %d", len(groups["fix"]))
	}
	if len(groups["other"]) != 1 {
		t.Fatalf("expected 1 other commit, got %d", len(groups["other"]))
	}
}

func TestGroup_FlatWhenNoneTyped(t *testing.T) {
	commits := []Commit{
		{Hash: "1", Subject: "updated readme"},
		{Hash: "2", Subject: "oops typo fix"},
	}

	_, _, flat := Group(commits)
	if !flat {
		t.Fatal("expected flat=true when no commits have a recognized conventional type")
	}
}

func TestRender_Grouped(t *testing.T) {
	commits := []Commit{
		{Hash: "1234567890", Subject: "feat(api): add endpoint"},
		{Hash: "abcdefabcdef", Subject: "fix: handle error"},
	}

	out := Render(commits, "v1.0.0", "HEAD")

	if !strings.Contains(out, "# Changelog (v1.0.0..HEAD)") {
		t.Fatalf("missing header, got:\n%s", out)
	}
	if !strings.Contains(out, "## Features") {
		t.Fatalf("missing Features heading, got:\n%s", out)
	}
	if !strings.Contains(out, "## Bug Fixes") {
		t.Fatalf("missing Bug Fixes heading, got:\n%s", out)
	}
	if !strings.Contains(out, "**api:** add endpoint (1234567)") {
		t.Fatalf("missing scoped feature entry, got:\n%s", out)
	}
	if !strings.Contains(out, "- handle error (abcdefa)") {
		t.Fatalf("missing fix entry, got:\n%s", out)
	}
}

func TestRender_Flat(t *testing.T) {
	commits := []Commit{
		{Hash: "1111111", Subject: "updated readme"},
		{Hash: "2222222", Subject: "typo fix"},
	}

	out := Render(commits, "abc", "def")
	if strings.Contains(out, "##") {
		t.Fatalf("flat changelog should not contain section headings, got:\n%s", out)
	}
	if !strings.Contains(out, "- updated readme (1111111)") {
		t.Fatalf("missing flat entry, got:\n%s", out)
	}
}

func TestRender_NoCommits(t *testing.T) {
	out := Render(nil, "a", "b")
	if !strings.Contains(out, "No commits found.") {
		t.Fatalf("expected no-commits message, got:\n%s", out)
	}
}

func TestParseGitLogOutput(t *testing.T) {
	raw := "hash1\x1ffeat: one\nhash2\x1ffix: two\n\nmalformed-line-without-separator\n"
	got := ParseGitLogOutput(raw)

	want := []Commit{
		{Hash: "hash1", Subject: "feat: one"},
		{Hash: "hash2", Subject: "fix: two"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d commits, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("commit %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
