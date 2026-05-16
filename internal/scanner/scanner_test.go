package scanner

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestScan(t *testing.T) {
	root := t.TempDir()

	// Plain repo: root/plain/.git/
	mkDir(t, filepath.Join(root, "plain", ".git"))

	// Linked worktree: root/worktree/.git as a file. A nested repo under it
	// must not be listed — the .git file should prune the walk.
	mkDir(t, filepath.Join(root, "worktree"))
	mkFile(t, filepath.Join(root, "worktree", ".git"), "gitdir: /elsewhere/.git/worktrees/wt")
	mkDir(t, filepath.Join(root, "worktree", "nested", ".git"))

	// Bare repo: root/bare.git/{HEAD, objects/, refs/}
	bare := filepath.Join(root, "bare.git")
	mkDir(t, bare)
	mkFile(t, filepath.Join(bare, "HEAD"), "ref: refs/heads/main\n")
	mkDir(t, filepath.Join(bare, "objects"))
	mkDir(t, filepath.Join(bare, "refs"))

	// Nested repo inside a regular repo: should be pruned, not listed.
	mkDir(t, filepath.Join(root, "outer", ".git"))
	mkDir(t, filepath.Join(root, "outer", "sub", ".git"))

	// Non-repo directory (sibling of repos, walked into and emerges empty).
	mkDir(t, filepath.Join(root, "notarepo", "deeper"))

	got, err := Scan([]string{root}, 10)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	want := []string{
		filepath.Join(root, "bare.git"),
		filepath.Join(root, "outer"),
		filepath.Join(root, "plain"),
	}
	if !slices.Equal(got, want) {
		t.Errorf("Scan returned %v, want %v", got, want)
	}
}

func TestScanRespectsMaxDepth(t *testing.T) {
	root := t.TempDir()
	mkDir(t, filepath.Join(root, "a", "b", "c", "repo", ".git"))

	// Repo is at depth 4 from root. maxDepth 3 should miss it.
	got, err := Scan([]string{root}, 3)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected no repos at maxDepth 3, got %v", got)
	}

	// maxDepth 4 should find it.
	got, err = Scan([]string{root}, 4)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	want := []string{filepath.Join(root, "a", "b", "c", "repo")}
	if !slices.Equal(got, want) {
		t.Errorf("Scan returned %v, want %v", got, want)
	}
}

func TestScanDeduplicatesAcrossRoots(t *testing.T) {
	parent := t.TempDir()
	mkDir(t, filepath.Join(parent, "repo", ".git"))

	got, err := Scan([]string{parent, parent}, 5)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	want := []string{filepath.Join(parent, "repo")}
	if !slices.Equal(got, want) {
		t.Errorf("expected dedup to %v, got %v", want, got)
	}
}

func TestScanDoesNotFollowSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks require elevation on Windows")
	}
	root := t.TempDir()
	target := t.TempDir()
	mkDir(t, filepath.Join(target, "hidden", ".git"))
	if err := os.Symlink(target, filepath.Join(root, "link")); err != nil {
		t.Fatalf("symlink: %v", err)
	}

	got, err := Scan([]string{root}, 10)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("symlink should not be followed, got %v", got)
	}
}

func TestScanSkipsInaccessibleDirs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod 0 semantics differ on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses permission checks")
	}
	root := t.TempDir()
	mkDir(t, filepath.Join(root, "ok", ".git"))
	blocked := filepath.Join(root, "blocked")
	mkDir(t, blocked)
	if err := os.Chmod(blocked, 0o000); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(blocked, 0o755) })

	got, err := Scan([]string{root}, 10)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	want := []string{filepath.Join(root, "ok")}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func mkDir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mkFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
