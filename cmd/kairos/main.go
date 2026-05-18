package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/shadmeoli/kairos/internal/config"
	"github.com/shadmeoli/kairos/internal/gitexec"
	"github.com/shadmeoli/kairos/internal/stashops"
	"github.com/shadmeoli/kairos/internal/store"
	"github.com/shadmeoli/kairos/internal/timeline"
	"github.com/shadmeoli/kairos/internal/ui"
	"github.com/shadmeoli/kairos/internal/version"
)

var (
	stashFlag bool
	labelFlag string
	noteFlag  string
	plainFlag bool
	startDir  string

	branchName     string
	preSwitchLabel string
	preSwitchNote  string

	stashPushMsg       string
	stashPushUntracked bool
)

func minArgs(cmd *cobra.Command, args []string) error {
	if branchName != "" && len(args) == 0 {
		return nil
	}
	if branchName == "" && len(args) >= 1 {
		return nil
	}
	return fmt.Errorf("use either `checkout <target>` or `checkout -b <branch>`")
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "kairos",
		Short: "Git time machine — save checkpoints and move along a timeline",
	}
	rootCmd.Version = version.Tag
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.PersistentFlags().StringVar(&startDir, "repo", ".", "path inside git working tree")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version (v:major.minor.patch)",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Print(version.Tag + "\n")
		},
	}

	saveCmd := &cobra.Command{
		Use:   "save",
		Short: "Record the current HEAD/branch as a checkpoint on the timeline",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			s, err := timeline.Save(repo, labelFlag, noteFlag)
			if err != nil {
				return err
			}
			printSavedCheckpoint(repo, s)
			return nil
		},
	}
	saveCmd.Flags().StringVar(&labelFlag, "label", "", "optional label for jump")
	saveCmd.Flags().StringVar(&noteFlag, "note", "", "optional note")

	checkoutCmd := &cobra.Command{
		Use:   "checkout [args]",
		Short: "Save current state, then git checkout (pass-through args)",
		Long:  "Records a checkpoint at your current HEAD/branch, then runs `git checkout` with the given arguments (e.g. a branch name or options git supports).",
		Args:  minArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			var gitArgs []string
			if branchName != "" {
				if len(args) > 0 {
					return fmt.Errorf("do not pass checkout args together with -b; use either `kairos checkout <target>` or `kairos checkout -b <branch>`")
				}
				gitArgs = []string{"checkout", "-b", branchName}
			} else {
				if len(args) == 0 {
					return fmt.Errorf("checkout target required")
				}
				gitArgs = append([]string{"checkout"}, args...)
			}
			return saveThenRunGit(repo, preSwitchLabel, preSwitchNote, gitArgs)
		},
	}
	checkoutCmd.Flags().StringVar(&preSwitchLabel, "label", "", "label for the checkpoint saved before checkout")
	checkoutCmd.Flags().StringVar(&preSwitchNote, "note", "", "note for the checkpoint saved before checkout")
	checkoutCmd.Flags().StringVarP(&branchName, "create", "b", "", "Initilize a new new branch and auto save it to the stack")

	switchCmd := &cobra.Command{
		Use:   "switch [args]",
		Short: "Save current state, then git switch (pass-through args)",
		Long:  "Records a checkpoint at your current HEAD/branch, then runs `git switch` with the given arguments (e.g. a branch name or -c / --detach as git allows).",
		Args:  minArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			var gitArgs []string
			if branchName != "" {
				if len(args) > 0 {
					return fmt.Errorf("do not pass switch args together with -b; use either `kairos switch <target>` or `kairos switch -c <branch>`")
				}
				gitArgs = []string{"switch", "-c", branchName}
			} else {
				if len(args) == 0 {
					return fmt.Errorf("checkout target required")
				}
				gitArgs = append([]string{"switch"}, args...)
			}
			return saveThenRunGit(repo, preSwitchLabel, preSwitchNote, gitArgs)
		},
	}
	switchCmd.Flags().StringVar(&preSwitchLabel, "label", "", "label for the checkpoint saved before switch")
	switchCmd.Flags().StringVar(&preSwitchNote, "note", "", "note for the checkpoint saved before switch")
	switchCmd.Flags().StringVarP(&branchName, "create", "c", "", "Create and switch to a new new branch and auto save it to the stack")

	stashCmd := &cobra.Command{
		Use:   "stash",
		Short: "Stash with extra metadata (parent branch, time, files) stored under .kairos/",
	}
	stashPushCmd := &cobra.Command{
		Use:   "push [pathspec...]",
		Short: "git stash push, then save kairos metadata for the new stash",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			return stashops.Push(repo, stashPushMsg, stashPushUntracked, args)
		},
	}
	stashPushCmd.Flags().StringVarP(&stashPushMsg, "message", "m", "", "stash message (passed to git)")
	stashPushCmd.Flags().BoolVarP(&stashPushUntracked, "include-untracked", "u", false, "pass --include-untracked to git stash push")

	stashListCmd := &cobra.Command{
		Use:   "list",
		Short: "git stash list with kairos metadata below each entry when available",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			return stashops.List(repo)
		},
	}
	stashShowCmd := &cobra.Command{
		Use:   "show [stash@{n}]",
		Short: "Print kairos metadata for a stash ref (default stash@{0})",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			ref := "stash@{0}"
			if len(args) == 1 {
				ref = args[0]
			}
			return stashops.Show(repo, ref)
		},
	}
	stashPopCmd := &cobra.Command{
		Use:   "pop [git stash pop args]",
		Short: "git stash pop; removes kairos metadata on success",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			return stashops.Pop(repo, args)
		},
	}
	stashApplyCmd := &cobra.Command{
		Use:   "apply [git stash apply args]",
		Short: "git stash apply (kairos metadata kept until pop)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			return stashops.Apply(repo, args)
		},
	}
	stashCmd.AddCommand(stashPushCmd, stashListCmd, stashShowCmd, stashPopCmd, stashApplyCmd)

	backCmd := &cobra.Command{
		Use:   "back",
		Short: "Move to the previous checkpoint (browser back)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			s, err := timeline.Back(repo, stashFlag)
			if err != nil {
				return err
			}
			printNav(repo, s)
			return nil
		},
	}
	forwardCmd := &cobra.Command{
		Use:   "forward",
		Short: "Move to the next checkpoint (browser forward)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			s, err := timeline.Forward(repo, stashFlag)
			if err != nil {
				return err
			}
			printNav(repo, s)
			return nil
		},
	}
	jumpCmd := &cobra.Command{
		Use:   "jump <label|index|id-prefix>",
		Short: "Jump to a checkpoint by index, label, or id prefix",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			s, err := timeline.Jump(repo, args[0], stashFlag)
			if err != nil {
				return err
			}
			printNav(repo, s)
			return nil
		},
	}
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show the timeline (TUI by default; use --plain for stdout graph)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			s, err := store.Load(repo)
			if err != nil {
				return err
			}
			head, err := gitexec.HEAD(gitexec.RepoRoot(repo))
			if err != nil {
				return err
			}
			if plainFlag {
				fmt.Print(renderPlainTree(repo, s, head))
				return nil
			}
			m := ui.NewTimelineModel(repo, s, head)
			p := tea.NewProgram(m)
			final, err := p.Run()
			if err != nil {
				return err
			}
			mod, ok := final.(ui.TimelineModel)
			if !ok {
				return nil
			}
			idx, chosen := mod.ChosenJump()
			if !chosen || idx < 0 {
				return nil
			}
			ns, err := timeline.Jump(repo, fmt.Sprintf("%d", idx), stashFlag)
			if err != nil {
				return err
			}
			printNav(repo, ns)
			return nil
		},
	}
	listCmd.Flags().BoolVar(&plainFlag, "plain", false, "print ASCII tree to stdout instead of TUI")
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show current git/timeline state",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			s, err := store.Load(repo)
			if err != nil {
				return err
			}
			head, err := gitexec.HEAD(gitexec.RepoRoot(repo))
			if err != nil {
				return err
			}
			branch, _ := gitexec.CurrentBranch(gitexec.RepoRoot(repo))
			detached, _ := gitexec.IsDetached(gitexec.RepoRoot(repo))
			clean, _ := gitexec.IsClean(gitexec.RepoRoot(repo))
			fmt.Printf("repo:     %s\n", repo)
			fmt.Printf("HEAD:     %s\n", head)
			if detached || branch == "" {
				fmt.Printf("branch:   (detached)\n")
			} else {
				fmt.Printf("branch:   %s\n", branch)
			}
			fmt.Printf("clean:    %v\n", clean)
			maxHist, err := config.EffectiveMaxHistory(repo)
			if err != nil {
				return err
			}
			fmt.Printf("timeline: %d/%d checkpoints (sliding window), cursor=%d\n", len(s.Checkpoints), maxHist, s.Cursor)
			if s.Cursor >= 0 && s.Cursor < len(s.Checkpoints) {
				cp := s.Checkpoints[s.Cursor]
				fmt.Printf("at:       [%d] %s @ %s\n", s.Cursor, shortRef(cp), shortHash(repo, cp.HEAD))
			}
			return nil
		},
	}

	for _, c := range []*cobra.Command{backCmd, forwardCmd, jumpCmd} {
		c.Flags().BoolVar(&stashFlag, "stash", false, "if dirty, stash changes before checkout (safer default is off)")
	}
	listCmd.Flags().BoolVar(&stashFlag, "stash", false, "when selecting a row in TUI, pass stash behavior to jump")

	configCmd := &cobra.Command{
		Use:   "config",
		Short: "View or set repo-local settings (.kairos/config.json)",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			max, explicit, err := config.Read(repo)
			if err != nil {
				return err
			}
			src := "default"
			if explicit {
				src = ".kairos/config.json"
			}
			fmt.Printf("max_history: %d (%s)\n", max, src)
			return nil
		},
	}
	maxHistoryCmd := &cobra.Command{
		Use:   "max-history [N]",
		Short: "Show or set sliding-window size for checkpoints",
		Long: fmt.Sprintf(
			"Allowed range: %d–%d. Default when unset: %d.",
			config.MinMaxHistory,
			config.MaxHistoryCap,
			config.DefaultMaxHistory,
		),
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := timeline.CurrentRepo(startDir)
			if err != nil {
				return err
			}
			if len(args) == 0 {
				max, explicit, err := config.Read(repo)
				if err != nil {
					return err
				}
				if explicit {
					fmt.Printf("max_history: %d (from .kairos/config.json)\n", max)
				} else {
					fmt.Printf("max_history: %d (default; set: kairos config max-history <n>)\n", max)
				}
				return nil
			}
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("max-history: not a number: %w", err)
			}
			if err := config.SetMaxHistory(repo, n); err != nil {
				return err
			}
			if err := store.EnsureGitignore(repo); err != nil {
				return err
			}
			if _, err := store.Load(repo); err != nil {
				return err
			}
			fmt.Printf("max_history=%d → .kairos/config.json (timeline re-trimmed if needed)\n", n)
			return nil
		},
	}
	configCmd.AddCommand(maxHistoryCmd)

	rootCmd.AddCommand(saveCmd, checkoutCmd, switchCmd, stashCmd, backCmd, forwardCmd, jumpCmd, listCmd, statusCmd, configCmd, versionCmd)

	rootCmd.Long = strings.TrimSpace(
		"Examples:\n" +
			"  kairos save --label before-refactor\n" +
			"  kairos checkout feature/x\n" +
			"  kairos switch main --label before-main\n" +
			"  kairos list              # interactive timeline; Enter jumps\n" +
			"  kairos list --plain      # print tree to stdout\n" +
			"  kairos back\n" +
			"  kairos forward\n" +
			"  kairos jump before-refactor\n" +
			"  kairos jump -1           # last checkpoint\n" +
			"  kairos status\n" +
			"  kairos config\n" +
			"  kairos config max-history 8\n" +
			"  kairos stash push -m wip -u\n" +
			"  kairos stash list\n" +
			"  kairos --repo ~/proj save --note wip",
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func validateCheckoutArgs(cmd *cobra.Command, args []string) error {
	if branchName != "" && len(args) == 0 {
		return nil
	}
	if branchName != "" && len(args) >= 1 {
		return nil
	}

	return fmt.Errorf("use either `checkout <target>` or `checkout -b <branch>`")
}

func printSavedCheckpoint(repo string, s store.State) {
	if s.Cursor < 0 || s.Cursor >= len(s.Checkpoints) {
		return
	}
	cp := s.Checkpoints[s.Cursor]
	fmt.Printf("saved checkpoint [%d] %s @ %s\n", s.Cursor, shortRef(cp), shortHash(repo, cp.HEAD))
}

func saveThenRunGit(repo string, label, note string, gitArgs []string) error {
	s, err := timeline.Save(repo, label, note)
	if err != nil {
		return err
	}
	printSavedCheckpoint(repo, s)
	return gitexec.RunInteractive(gitexec.RepoRoot(repo), gitArgs)
}

func printNav(repo string, s store.State) {
	if s.Cursor < 0 || s.Cursor >= len(s.Checkpoints) {
		return
	}
	cp := s.Checkpoints[s.Cursor]
	fmt.Printf("now at [%d] %s @ %s\n", s.Cursor, shortRef(cp), shortHash(repo, cp.HEAD))
}

func shortRef(cp store.Checkpoint) string {
	if cp.Detached || cp.Branch == "" {
		return "detached"
	}
	return cp.Branch
}

func shortHash(repo, rev string) string {
	h, err := gitexec.ShortHash(gitexec.RepoRoot(repo), rev)
	if err != nil {
		if len(rev) >= 7 {
			return rev[:7]
		}
		return rev
	}
	return h
}

func renderPlainTree(repo string, s store.State, head string) string {
	var b strings.Builder
	root := gitexec.RepoRoot(repo)
	for i := len(s.Checkpoints) - 1; i >= 0; i-- {
		cp := s.Checkpoints[i]
		short, _ := gitexec.ShortHash(root, cp.HEAD)
		if short == "" && len(cp.HEAD) >= 7 {
			short = cp.HEAD[:7]
		}
		ref := cp.Branch
		if cp.Detached || ref == "" {
			ref = "detached"
		}
		prefix := "│ "
		if i == len(s.Checkpoints)-1 {
			prefix = "* "
		}
		tag := ""
		if i == s.Cursor {
			tag += "  ← cursor"
		}
		if strings.HasPrefix(head, cp.HEAD) || head == cp.HEAD {
			tag += "  [HEAD]"
		}
		label := ""
		if cp.Label != "" {
			label = fmt.Sprintf(" (%s)", cp.Label)
		}
		fmt.Fprintf(&b, "%s[%d] %-14s @ %s%s%s\n", prefix, i, ref, short, label, tag)
	}
	if len(s.Checkpoints) == 0 {
		b.WriteString("(no checkpoints)\n")
	}
	return b.String()
}
