// Package changelog builds a simple changelog from a list of git commits.
// When commit subjects follow the Conventional Commits style
// (`type(scope): description`), entries are grouped by type; otherwise a
// flat list is produced.
package changelog

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"
)

// Commit is a single git log entry.
type Commit struct {
	Hash    string
	Subject string
}

// ParsedCommit is a Commit with its conventional-commit parts extracted,
// when recognized.
type ParsedCommit struct {
	Commit
	Type        string
	Scope       string
	Breaking    bool
	Description string
}

// knownTypes lists the conventional-commit type prefixes octo recognizes,
// in the order they should appear in a generated changelog.
var knownTypes = []string{"feat", "fix", "perf", "refactor", "docs", "style", "test", "build", "ci", "chore"}

var conventionalRe = regexp.MustCompile(`^([a-zA-Z]+)(\([^)]*\))?(!)?:\s*(.+)$`)

var typeHeadings = map[string]string{
	"feat":     "Features",
	"fix":      "Bug Fixes",
	"perf":     "Performance",
	"refactor": "Refactoring",
	"docs":     "Documentation",
	"style":    "Styles",
	"test":     "Tests",
	"build":    "Build System",
	"ci":       "Continuous Integration",
	"chore":    "Chores",
	"other":    "Other Changes",
}

func isKnownType(t string) bool {
	for _, k := range knownTypes {
		if k == t {
			return true
		}
	}
	return false
}

// ParseCommit extracts conventional-commit metadata from a commit subject.
// If the subject doesn't match the `type(scope)!: description` pattern, or
// the type isn't recognized, Type is left empty and Description is the
// original subject.
func ParseCommit(c Commit) ParsedCommit {
	m := conventionalRe.FindStringSubmatch(c.Subject)
	if m == nil {
		return ParsedCommit{Commit: c, Description: c.Subject}
	}
	typ := strings.ToLower(m[1])
	if !isKnownType(typ) {
		return ParsedCommit{Commit: c, Description: c.Subject}
	}
	scope := strings.TrimSuffix(strings.TrimPrefix(m[2], "("), ")")
	return ParsedCommit{
		Commit:      c,
		Type:        typ,
		Scope:       scope,
		Breaking:    m[3] == "!",
		Description: m[4],
	}
}

// Group splits commits into buckets keyed by conventional-commit type. If
// none of the commits have a recognized type, flat is true and the caller
// should render commits as a flat list instead.
func Group(commits []Commit) (groups map[string][]ParsedCommit, order []string, flat bool) {
	parsed := make([]ParsedCommit, len(commits))
	anyTyped := false
	for i, c := range commits {
		p := ParseCommit(c)
		parsed[i] = p
		if p.Type != "" {
			anyTyped = true
		}
	}
	if !anyTyped {
		return nil, nil, true
	}

	groups = make(map[string][]ParsedCommit)
	for _, p := range parsed {
		key := p.Type
		if key == "" {
			key = "other"
		}
		groups[key] = append(groups[key], p)
	}

	for _, t := range knownTypes {
		if _, ok := groups[t]; ok {
			order = append(order, t)
		}
	}
	if _, ok := groups["other"]; ok {
		order = append(order, "other")
	}
	return groups, order, false
}

func headingFor(t string) string {
	if h, ok := typeHeadings[t]; ok {
		return h
	}
	return strings.ToUpper(t[:1]) + t[1:]
}

func shortHash(h string) string {
	if len(h) > 7 {
		return h[:7]
	}
	return h
}

// Render produces a markdown changelog for the given commits, covering the
// range from..to.
func Render(commits []Commit, from, to string) string {
	// Sanitize before parsing so both flat subjects and conventional scopes and
	// descriptions are safe. Copy to leave the caller's commits unchanged.
	safe := make([]Commit, len(commits))
	for i, c := range commits {
		c.Subject = strings.Map(func(r rune) rune {
			if unicode.IsControl(r) {
				return -1
			}
			return r
		}, c.Subject)
		safe[i] = c
	}
	commits = safe
	var sb strings.Builder
	fmt.Fprintf(&sb, "# Changelog (%s..%s)\n\n", from, to)

	if len(commits) == 0 {
		sb.WriteString("No commits found.\n")
		return sb.String()
	}

	groups, order, flat := Group(commits)
	if flat {
		for _, c := range commits {
			fmt.Fprintf(&sb, "- %s (%s)\n", c.Subject, shortHash(c.Hash))
		}
		return sb.String()
	}

	for _, t := range order {
		fmt.Fprintf(&sb, "## %s\n\n", headingFor(t))
		for _, p := range groups[t] {
			desc := p.Description
			if p.Breaking {
				desc = "**BREAKING:** " + desc
			}
			if p.Scope != "" {
				fmt.Fprintf(&sb, "- **%s:** %s (%s)\n", p.Scope, desc, shortHash(p.Hash))
			} else {
				fmt.Fprintf(&sb, "- %s (%s)\n", desc, shortHash(p.Hash))
			}
		}
		sb.WriteString("\n")
	}
	return strings.TrimRight(sb.String(), "\n") + "\n"
}

// logSeparator delimits the commit hash from the subject in the `git log`
// format string octo requests (see cli.newChangelogCmd).
const logSeparator = "\x1f"

// ParseGitLogOutput parses output produced by
// `git log --format=%H` + logSeparator + `%s` into a slice of Commits.
func ParseGitLogOutput(output string) []Commit {
	var commits []Commit
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.SplitN(line, logSeparator, 2)
		if len(parts) != 2 {
			continue
		}
		commits = append(commits, Commit{Hash: parts[0], Subject: parts[1]})
	}
	return commits
}

// LogFormat is the `git log` --format argument octo uses so ParseGitLogOutput
// can split hash and subject reliably.
const LogFormat = "%H" + logSeparator + "%s"
