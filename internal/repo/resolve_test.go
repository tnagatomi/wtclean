package repo

import (
	"os"
	"path/filepath"
	"testing"
)

// resolved returns the symlink-resolved form of path so comparisons hold on
// platforms (macOS) where the temp dir lives behind a /var → /private/var
// symlink while git reports the realpath.
func resolved(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", path, err)
	}
	return abs
}

func TestResolvePrimaryDir_FromPrimary(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	primary := filepath.Join(root, "repo")
	initRepo(t, primary)

	got, err := ResolvePrimaryDir(primary)
	if err != nil {
		t.Fatalf("ResolvePrimaryDir: %v", err)
	}
	if got != resolved(t, primary) {
		t.Errorf("got %q, want %q", got, resolved(t, primary))
	}
}

func TestResolvePrimaryDir_FromLinkedWorktree(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	primary := filepath.Join(root, "repo")
	initRepo(t, primary)
	linked := filepath.Join(root, "repo-feat")
	addWorktree(t, primary, linked, "feat")

	got, err := ResolvePrimaryDir(linked)
	if err != nil {
		t.Fatalf("ResolvePrimaryDir: %v", err)
	}
	if got != resolved(t, primary) {
		t.Errorf("got %q, want %q (must return primary, not the linked worktree)", got, resolved(t, primary))
	}
}

func TestResolvePrimaryDir_NotInRepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir() // a bare temp dir, never git init'd

	if _, err := ResolvePrimaryDir(dir); err == nil {
		t.Fatalf("expected an error when dir is not inside a git repository")
	}
}

func TestResolvePrimaryDir_FromSubdir(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	primary := filepath.Join(root, "repo")
	initRepo(t, primary)
	sub := filepath.Join(primary, "pkg", "deep")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	got, err := ResolvePrimaryDir(sub)
	if err != nil {
		t.Fatalf("ResolvePrimaryDir: %v", err)
	}
	if got != resolved(t, primary) {
		t.Errorf("got %q, want %q", got, resolved(t, primary))
	}
}
