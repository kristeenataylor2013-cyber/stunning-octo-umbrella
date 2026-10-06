package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/kristeenataylor2013-cyber/stunning-octo-umbrella/internal/gitutil"
)

func newCleanCmd() *cobra.Command {
	var dryRun bool
	var force bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Delete local branches already merged into the default branch",
		Long: `clean scans the current git repository for local branches that have
already been merged into the default branch (main/master, or whatever
origin/HEAD points at) and removes them.

By default it only prints what would be deleted. Pass --force to actually
delete the branches.`,
		Example: `  octo clean            # dry-run: show merged branches
  octo clean --force    # actually delete them`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			effectiveDryRun := dryRun
			if force && !cmd.Flags().Changed("dry-run") {
				effectiveDryRun = false
			}
			return runClean(cmd, gitutil.ExecRunner{}, effectiveDryRun, force)
		},
	}

	cmd.Flags().BoolVar(&dryRun, "dry-run", true, "show branches that would be deleted without deleting them")
	cmd.Flags().BoolVar(&force, "force", false, "actually delete the merged branches")
	return cmd
}

func runClean(cmd *cobra.Command, runner gitutil.Runner, dryRun bool, force bool) error {
	out := cmd.OutOrStdout()

	defaultBranch, err := gitutil.DetectDefaultBranch(runner)
	if err != nil {
		return err
	}

	current, err := gitutil.CurrentBranch(runner)
	if err != nil {
		return err
	}

	branches, err := gitutil.ListMergedBranches(runner, defaultBranch)
	if err != nil {
		return err
	}

	filtered := branches[:0:0]
	for _, b := range branches {
		if b == current {
			continue
		}
		filtered = append(filtered, b)
	}

	if len(filtered) == 0 {
		fmt.Fprintf(out, "No local branches merged into %q to clean up.\n", defaultBranch)
		return nil
	}

	if dryRun || !force {
		fmt.Fprintf(out, "The following branches are merged into %q and would be deleted (dry-run):\n", defaultBranch)
		for _, b := range filtered {
			fmt.Fprintf(out, "  - %s\n", b)
		}
		fmt.Fprintln(out, "Re-run with --force to delete them.")
		return nil
	}

	deleted, errs := gitutil.DeleteBranches(runner, filtered, force)
	for _, b := range deleted {
		fmt.Fprintf(out, "Deleted %s\n", b)
	}
	for _, e := range errs {
		fmt.Fprintf(cmd.ErrOrStderr(), "error: %v\n", e)
	}
	if len(errs) > 0 {
		return fmt.Errorf("%d branch(es) failed to delete", len(errs))
	}
	return nil
}
