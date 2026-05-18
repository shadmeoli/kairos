package gitexec

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type RepoRoot string

func Run(repo RepoRoot, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if repo != "" {
		cmd.Dir = string(repo)
	}
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %v: %w: %s", args, err, msg)
	}
	return strings.TrimSpace(out.String()), nil
}

func CreateBranch(repo string, branchName string) bool {
	out, err := Run("", "branch", "--create", branchName)
	if err != nil {
		fmt.Printf("Unable to create branch\n%s", err)
		return false
	}

	if out == "" {
		return false
	}
	fmt.Printf("Branch created: %s", out)
	return true

}

func RepoTopLevel(startDir string) (string, error) {
	out, err := Run("", "-C", startDir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	if out == "" {
		return "", errors.New("empty git top-level")
	}
	return out, nil
}

func HEAD(repo RepoRoot) (string, error) {
	return Run(repo, "rev-parse", "HEAD")
}

func IsDetached(repo RepoRoot) (bool, error) {
	abbr, err := Run(repo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return false, err
	}
	return abbr == "HEAD", nil
}

func CurrentBranch(repo RepoRoot) (string, error) {
	ab, err := Run(repo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if ab == "HEAD" {
		return "", nil
	}
	return ab, nil
}

func IsClean(repo RepoRoot) (bool, error) {
	out, err := Run(repo, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) == "", nil
}

func StashPush(repo RepoRoot, message string) error {
	_, err := Run(repo, "stash", "push", "-u", "-m", message)
	return err
}

func Checkout(repo RepoRoot, target string, new bool) error {
	_, err := Run(repo, "checkout", "--quiet", target)
	return err
}

func ShortHash(repo RepoRoot, rev string) (string, error) {
	if rev == "" {
		return "", errors.New("empty rev")
	}
	return Run(repo, "rev-parse", "--short", rev)
}

// RunInteractive runs git with stdout/stderr attached (for checkout/switch).
func StashShowFilenames(repo RepoRoot, stashRef string, wantUntracked bool) ([]string, error) {
	try := func(untracked bool) (string, error) {
		args := []string{"stash", "show", "--name-only"}
		if untracked {
			args = append(args, "--include-untracked")
		}
		args = append(args, stashRef)
		return Run(repo, args...)
	}
	out, err := try(wantUntracked)
	if err != nil && wantUntracked {
		out, err = try(false)
	}
	if err != nil {
		return nil, err
	}
	out = strings.TrimSpace(out)
	if out == "" {
		return nil, nil
	}
	var names []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			names = append(names, line)
		}
	}
	return names, nil
}

// StashLogSubject returns the one-line subject of the stash commit.
func StashLogSubject(repo RepoRoot, stashRef string) (string, error) {
	return Run(repo, "log", "-1", "--format=%s", stashRef)
}

func RunInteractive(repo RepoRoot, gitArgs []string) error {
	if len(gitArgs) == 0 {
		return errors.New("no git arguments")
	}
	cmd := exec.Command("git", gitArgs...)
	if repo != "" {
		cmd.Dir = string(repo)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %v: %w", gitArgs, err)
	}
	return nil
}
