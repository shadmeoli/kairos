package gitexec

import (
	"bytes"
	"errors"
	"fmt"
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

func Checkout(repo RepoRoot, target string) error {
	_, err := Run(repo, "checkout", "--quiet", target)
	return err
}

func ShortHash(repo RepoRoot, rev string) (string, error) {
	if rev == "" {
		return "", errors.New("empty rev")
	}
	return Run(repo, "rev-parse", "--short", rev)
}
