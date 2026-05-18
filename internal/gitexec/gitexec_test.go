package gitexec

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepoTopLevel(t *testing.T) {
	repo := initTestRepo(t)
	nested := filepath.Join(repo, "nested", "dir")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested dir: %v", err)
	}

	got, err := RepoTopLevel(nested)
	if err != nil {
		t.Fatalf("RepoTopLevel() error = %v", err)
	}
	if got != repo {
		t.Fatalf("RepoTopLevel() = %q, want %q", got, repo)
	}
}

func TestBranchStateOnAttachedHEAD(t *testing.T) {
	repo := initTestRepo(t)
	root := RepoRoot(repo)

	head, err := HEAD(root)
	if err != nil {
		t.Fatalf("HEAD() error = %v", err)
	}
	if len(head) != 40 {
		t.Fatalf("HEAD() length = %d, want 40", len(head))
	}

	branch, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if branch != "main" {
		t.Fatalf("CurrentBranch() = %q, want %q", branch, "main")
	}

	detached, err := IsDetached(root)
	if err != nil {
		t.Fatalf("IsDetached() error = %v", err)
	}
	if detached {
		t.Fatalf("IsDetached() = true, want false")
	}
}

func TestBranchStateOnDetachedHEAD(t *testing.T) {
	repo := initTestRepo(t)
	root := RepoRoot(repo)

	head, err := HEAD(root)
	if err != nil {
		t.Fatalf("HEAD() error = %v", err)
	}
	runGit(t, repo, "checkout", "--detach", head)

	branch, err := CurrentBranch(root)
	if err != nil {
		t.Fatalf("CurrentBranch() error = %v", err)
	}
	if branch != "" {
		t.Fatalf("CurrentBranch() = %q, want empty string", branch)
	}

	detached, err := IsDetached(root)
	if err != nil {
		t.Fatalf("IsDetached() error = %v", err)
	}
	if !detached {
		t.Fatalf("IsDetached() = false, want true")
	}
}

func TestIsCleanTracksWorkingTreeChanges(t *testing.T) {
	repo := initTestRepo(t)
	root := RepoRoot(repo)

	clean, err := IsClean(root)
	if err != nil {
		t.Fatalf("IsClean() error = %v", err)
	}
	if !clean {
		t.Fatalf("IsClean() = false, want true")
	}

	dirtyFile := filepath.Join(repo, "dirty.txt")
	if err := os.WriteFile(dirtyFile, []byte("change\n"), 0o644); err != nil {
		t.Fatalf("write dirty file: %v", err)
	}

	clean, err = IsClean(root)
	if err != nil {
		t.Fatalf("IsClean() after change error = %v", err)
	}
	if clean {
		t.Fatalf("IsClean() = true, want false")
	}
}

func TestShortHash(t *testing.T) {
	repo := initTestRepo(t)
	root := RepoRoot(repo)

	head, err := HEAD(root)
	if err != nil {
		t.Fatalf("HEAD() error = %v", err)
	}

	short, err := ShortHash(root, head)
	if err != nil {
		t.Fatalf("ShortHash() error = %v", err)
	}
	if short == "" {
		t.Fatalf("ShortHash() returned empty string")
	}
	if len(short) >= len(head) {
		t.Fatalf("ShortHash() length = %d, want less than %d", len(short), len(head))
	}
}

func TestCreateBranchUsesProvidedRepo(t *testing.T) {
	repo := initTestRepo(t)

	if ok := CreateBranch(repo, "feature/test"); !ok {
		t.Fatalf("CreateBranch() = false, want true")
	}

	out := runGit(t, repo, "branch", "--list", "feature/test")
	if !strings.Contains(out, "feature/test") {
		t.Fatalf("created branch not found in git branch output: %q", out)
	}
}

func initTestRepo(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	runGit(t, repo, "init", "-b", "main")
	runGit(t, repo, "config", "user.name", "Kairos Tests")
	runGit(t, repo, "config", "user.email", "kairos-tests@example.com")

	readme := filepath.Join(repo, "README.md")
	if err := os.WriteFile(readme, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write seed file: %v", err)
	}

	runGit(t, repo, "add", "README.md")
	runGit(t, repo, "commit", "-m", "initial commit")
	return repo
}

func runGit(t *testing.T, repo string, args ...string) string {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = repo
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out))
}
