package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/internal/changelog"
	"github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/internal/gitutil"
)

func newChangelogCmd() *cobra.Command {
	var from, to, output string

	cmd := &cobra.Command{
		Use:   "changelog",
		Short: "Generate a changelog from git log between two refs",
		Long: `changelog generates a simple changelog from "git log" between two refs.

Commit subjects that follow the Conventional Commits style
("feat: ...", "fix(scope): ...", etc.) are grouped by type; otherwise a flat
list of commits is produced.`,
		Example: `  octo changelog
  octo changelog --from v1.0.0 --to HEAD
  octo changelog --from v1.0.0 --to v1.1.0 --output CHANGELOG.md`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runChangelog(cmd, gitutil.ExecRunner{}, from, to, output)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "starting ref (defaults to the latest tag, or the first commit if there are no tags)")
	cmd.Flags().StringVar(&to, "to", "HEAD", "ending ref")
	cmd.Flags().StringVar(&output, "output", "", "write the changelog to this file instead of stdout")
	return cmd
}

func runChangelog(cmd *cobra.Command, runner gitutil.Runner, from, to, output string) error {
	resolvedTo := to
	if resolvedTo == "" {
		resolvedTo = "HEAD"
	}

	resolvedFrom := from
	if resolvedFrom == "" {
		if tag, err := gitutil.LatestTag(runner); err == nil && tag != "" {
			resolvedFrom = tag
		} else if first, err := gitutil.FirstCommit(runner); err == nil {
			resolvedFrom = first
		} else {
			return fmt.Errorf("could not determine a starting ref: %w", err)
		}
	}

	rangeArg := fmt.Sprintf("%s..%s", resolvedFrom, resolvedTo)
	out, err := runner.Run("log", rangeArg, "--format="+changelog.LogFormat)
	if err != nil {
		return fmt.Errorf("git log failed: %w", err)
	}

	commits := changelog.ParseGitLogOutput(out)
	text := changelog.Render(commits, resolvedFrom, resolvedTo)

	if output != "" {
		if err := os.WriteFile(output, []byte(text), 0o644); err != nil {
			return fmt.Errorf("write changelog to %s: %w", output, err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Changelog written to %s\n", output)
		return nil
	}

	fmt.Fprint(cmd.OutOrStdout(), text)
	return nil
}
