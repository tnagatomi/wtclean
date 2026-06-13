package cli

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func initRepo(t *testing.T, path string) {
	t.Helper()
	mustRun(t, "git", "init", "-q", "-b", "main", path)
	mustRun(t, "git", "-C", path, "config", "user.email", "test@example.com")
	mustRun(t, "git", "-C", path, "config", "user.name", "Test")
	mustRun(t, "git", "-C", path, "commit", "--allow-empty", "-q", "-m", "init")
}

func addWorktree(t *testing.T, repoPath, wtPath, branch string) {
	t.Helper()
	mustRun(t, "git", "-C", repoPath, "worktree", "add", "-q", "-b", branch, wtPath)
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}

func TestRootHasLocalCwdFlag(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Flags().Lookup("cwd") == nil {
		t.Errorf("root command should define a --cwd flag")
	}
	// Local, not persistent: --cwd must not be inherited by subcommands
	// such as `init`, for which it is meaningless.
	if cmd.PersistentFlags().Lookup("cwd") != nil {
		t.Errorf("--cwd should be a local flag, not a persistent (inherited) one")
	}
}

func TestNewCwdModelErrorsOutsideRepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir() // never git init'd
	if _, err := newCwdModel(dir); err == nil {
		t.Fatalf("newCwdModel should error when dir is not inside a git repository")
	}
}

func TestNewCwdModelOpensWorktreeScreenInRepo(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	primary := filepath.Join(root, "repo")
	initRepo(t, primary)
	addWorktree(t, primary, filepath.Join(root, "repo-feat"), "feat-x")

	// Resolve from a linked worktree to prove resolution walks to the primary.
	m, err := newCwdModel(filepath.Join(root, "repo-feat"))
	if err != nil {
		t.Fatalf("newCwdModel: %v", err)
	}
	view := m.View().Content
	if !strings.Contains(view, "worktrees in") {
		t.Errorf("model should open directly on the worktree screen: %q", view)
	}
	if !strings.Contains(view, "feat-x") {
		t.Errorf("worktree screen should list the repo's linked worktree: %q", view)
	}
}
