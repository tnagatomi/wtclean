package fetcher

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestFetchSucceedsAgainstLocalRemote(t *testing.T) {
	requireGit(t)
	// Set up: a "remote" bare repo with one commit, and a "local" clone
	// of it. The local should be able to fetch from origin successfully.
	dir := t.TempDir()
	remote := filepath.Join(dir, "remote.git")
	source := filepath.Join(dir, "source")
	local := filepath.Join(dir, "local")
	mustRun(t, "git", "init", "-q", "-b", "main", source)
	mustRun(t, "git", "-C", source, "config", "user.email", "test@example.com")
	mustRun(t, "git", "-C", source, "config", "user.name", "Test")
	mustRun(t, "git", "-C", source, "commit", "--allow-empty", "-q", "-m", "init")
	mustRun(t, "git", "clone", "-q", "--bare", source, remote)
	mustRun(t, "git", "clone", "-q", remote, local)

	if err := Fetch(local); err != nil {
		t.Fatalf("Fetch against a healthy clone should succeed: %v", err)
	}
}

func TestFetchReturnsErrorIncludingStderrWhenRepoMissing(t *testing.T) {
	requireGit(t)
	err := Fetch("/nonexistent/path/for/fetcher/test")
	if err == nil {
		t.Fatal("Fetch against a missing repo should return an error")
	}
	// We expect the error to include git's diagnostic text, not just an
	// opaque exec error. The exact wording can change between git
	// versions, so just assert that "not a git repository" or some
	// recognizable substring of git's output is present.
	low := strings.ToLower(err.Error())
	if !strings.Contains(low, "not a git repository") && !strings.Contains(low, "no such file") {
		t.Errorf("error should include git's diagnostic; got: %v", err)
	}
}

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

func mustRun(t *testing.T, name string, args ...string) {
	t.Helper()
	if out, err := exec.Command(name, args...).CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
