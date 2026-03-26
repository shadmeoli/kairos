package stashops

import (
	"fmt"
	"strings"
	"time"

	"github.com/shadmeoli/kairos/internal/gitexec"
	"github.com/shadmeoli/kairos/internal/stashmeta"
)

func formatParent(e *stashmeta.Entry) string {
	if !e.Detached && e.ParentBranch != "" {
		return e.ParentBranch
	}
	sh := e.ParentHEAD
	if len(sh) >= 7 {
		sh = sh[:7]
	}
	if sh == "" {
		return "(detached)"
	}
	return fmt.Sprintf("(detached @ %s)", sh)
}

func printEntry(e *stashmeta.Entry) {
	fmt.Printf("  kairos: parent %s | %s | %d file(s)\n", formatParent(e), e.CreatedAt.Format(time.RFC3339), len(e.Files))
	for _, f := range e.Files {
		fmt.Printf("    %s\n", f)
	}
}

// Push runs git stash push, then records kairos metadata for the new stash@{0}.
func Push(repoTop string, message string, includeUntracked bool, pathspecs []string) error {
	root := gitexec.RepoRoot(repoTop)
	detached, err := gitexec.IsDetached(root)
	if err != nil {
		return err
	}
	branch, err := gitexec.CurrentBranch(root)
	if err != nil {
		return err
	}
	head, err := gitexec.HEAD(root)
	if err != nil {
		return err
	}

	gitArgs := []string{"stash", "push"}
	if includeUntracked {
		gitArgs = append(gitArgs, "--include-untracked")
	}
	if message != "" {
		gitArgs = append(gitArgs, "-m", message)
	}
	if len(pathspecs) > 0 {
		gitArgs = append(gitArgs, "--")
		gitArgs = append(gitArgs, pathspecs...)
	}
	created := time.Now().UTC()
	if err := gitexec.RunInteractive(root, gitArgs); err != nil {
		return err
	}

	stashRef := "stash@{0}"
	sha, err := gitexec.Run(root, "rev-parse", stashRef)
	if err != nil {
		return fmt.Errorf("after stash: %w", err)
	}
	files, err := gitexec.StashShowFilenames(root, stashRef, includeUntracked)
	if err != nil {
		return err
	}
	msg := message
	if msg == "" {
		msg, _ = gitexec.StashLogSubject(root, stashRef)
	}

	entry := stashmeta.Entry{
		StashSHA:     sha,
		ParentBranch: branch,
		Detached:     detached,
		ParentHEAD:   head,
		CreatedAt:    created,
		Files:        files,
		Message:      msg,
	}
	return stashmeta.Upsert(repoTop, entry)
}

// Pop runs git stash pop and drops kairos metadata when it succeeds.
func Pop(repoTop string, gitArgs []string) error {
	root := gitexec.RepoRoot(repoTop)
	ref := "stash@{0}"
	for _, a := range gitArgs {
		if strings.HasPrefix(a, "stash@{") {
			ref = a
			break
		}
	}
	sha, err := gitexec.Run(root, "rev-parse", ref)
	if err != nil {
		return err
	}
	args := append([]string{"stash", "pop"}, gitArgs...)
	if err := gitexec.RunInteractive(root, args); err != nil {
		return err
	}
	_ = stashmeta.Delete(repoTop, sha)
	return nil
}

// List prints git stash list plus kairos metadata when present.
func List(repoTop string) error {
	root := gitexec.RepoRoot(repoTop)
	out, err := gitexec.Run(root, "stash", "list")
	if err != nil {
		return err
	}
	if strings.TrimSpace(out) == "" {
		return nil
	}
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		ref := fmt.Sprintf("stash@{%d}", i)
		fmt.Println(line)
		sha, err := gitexec.Run(root, "rev-parse", ref)
		if err != nil {
			continue
		}
		if e, ok := stashmeta.Get(repoTop, sha); ok {
			printEntry(&e)
		}
	}
	return nil
}

// Show prints kairos metadata for a stash ref (e.g. stash@{0}).
func Show(repoTop, stashRef string) error {
	root := gitexec.RepoRoot(repoTop)
	sha, err := gitexec.Run(root, "rev-parse", stashRef)
	if err != nil {
		return err
	}
	e, ok := stashmeta.Get(repoTop, sha)
	if !ok {
		return fmt.Errorf("no kairos metadata for %s (only stashes created with kairos stash push are recorded)", stashRef)
	}
	printEntry(&e)
	return nil
}

// Apply runs git stash apply (metadata kept until pop).
func Apply(repoTop string, gitArgs []string) error {
	root := gitexec.RepoRoot(repoTop)
	args := append([]string{"stash", "apply"}, gitArgs...)
	return gitexec.RunInteractive(root, args)
}
