package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/tnagatomi/wtm/internal/worktree"
)

func TestDiscover(t *testing.T) {
	requireGit(t)
	root := t.TempDir()

	// Repo A: primary plus one linked worktree → should appear.
	repoA := filepath.Join(root, "a")
	initRepo(t, repoA)
	addWorktree(t, repoA, filepath.Join(root, "a-wt"), "feat-a")

	// Repo B: primary only → should be hidden.
	repoB := filepath.Join(root, "b")
	initRepo(t, repoB)

	// Repo C: primary plus two linked → should appear.
	repoC := filepath.Join(root, "c")
	initRepo(t, repoC)
	addWorktree(t, repoC, filepath.Join(root, "c-wt1"), "feat-c1")
	addWorktree(t, repoC, filepath.Join(root, "c-wt2"), "feat-c2")

	repos, err := Discover([]string{root}, 5)
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(repos) != 2 {
		t.Fatalf("want 2 repos with linked worktrees, got %d: %+v", len(repos), repos)
	}
	if repos[0].Path != repoA || repos[1].Path != repoC {
		t.Errorf("repos not sorted by path: got %s, %s", repos[0].Path, repos[1].Path)
	}
	if repos[0].LinkedCount() != 1 {
		t.Errorf("repo a: want 1 linked, got %d", repos[0].LinkedCount())
	}
	if repos[1].LinkedCount() != 2 {
		t.Errorf("repo c: want 2 linked, got %d", repos[1].LinkedCount())
	}
}

func TestLinkedCount(t *testing.T) {
	cases := []struct {
		name  string
		count int
		want  int
	}{
		{"empty", 0, 0},
		{"primary only", 1, 0},
		{"primary plus one linked", 2, 1},
		{"primary plus three linked", 4, 3},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := Repo{Worktrees: make([]worktree.Worktree, c.count)}
			if got := r.LinkedCount(); got != c.want {
				t.Errorf("got %d, want %d", got, c.want)
			}
		})
	}
}

func TestFetchHeadMtime(t *testing.T) {
	t.Run("zero for never-fetched", func(t *testing.T) {
		requireGit(t)
		dir := t.TempDir()
		initRepo(t, dir)
		if got := fetchHeadMtime(dir); !got.IsZero() {
			t.Errorf("expected zero time, got %v", got)
		}
	})

	t.Run("returns mtime when FETCH_HEAD exists", func(t *testing.T) {
		requireGit(t)
		dir := t.TempDir()
		initRepo(t, dir)
		fh := filepath.Join(dir, ".git", "FETCH_HEAD")
		if err := os.WriteFile(fh, []byte("fake fetch\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		got := fetchHeadMtime(dir)
		if got.IsZero() {
			t.Fatalf("expected non-zero time")
		}
		if time.Since(got) > time.Minute {
			t.Errorf("mtime should be recent, got %v", got)
		}
	})
}

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
