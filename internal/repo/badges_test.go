package repo

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/tnagatomi/wtm/internal/worktree"
)

func TestPopulateBadgesPrimary(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "r")
	initRepo(t, repoPath)
	addWorktree(t, repoPath, filepath.Join(dir, "wt"), "feat")

	r, err := Load(repoPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !hasBadge(r.Worktrees[0], worktree.BadgePrimary) {
		t.Errorf("first worktree should carry primary badge: %+v", r.Worktrees[0].Badges)
	}
	if hasBadge(r.Worktrees[1], worktree.BadgePrimary) {
		t.Errorf("linked worktree should not carry primary badge: %+v", r.Worktrees[1].Badges)
	}
}

func TestPopulateBadgesMerged(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "r")
	initRepo(t, repoPath)
	addWorktree(t, repoPath, filepath.Join(dir, "wt"), "merged-branch")
	// merged-branch was created from main and has no new commits, so
	// `git branch --merged main` lists it.

	r, err := Load(repoPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	linked := r.Worktrees[1]
	if !hasBadge(linked, worktree.BadgeMerged) {
		t.Errorf("merged-branch should carry merged badge: %+v", linked.Badges)
	}
}

func TestPopulateBadgesDirty(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "r")
	initRepo(t, repoPath)
	wtPath := filepath.Join(dir, "wt")
	addWorktree(t, repoPath, wtPath, "feat")
	// Introduce an untracked file in the linked worktree.
	if err := os.WriteFile(filepath.Join(wtPath, "junk.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	r, err := Load(repoPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !hasBadge(r.Worktrees[1], worktree.BadgeDirty) {
		t.Errorf("worktree with untracked file should be dirty: %+v", r.Worktrees[1].Badges)
	}
}

func TestPopulateBadgesLocked(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "r")
	initRepo(t, repoPath)
	wtPath := filepath.Join(dir, "wt")
	addWorktree(t, repoPath, wtPath, "feat")
	mustRun(t, "git", "-C", repoPath, "worktree", "lock", wtPath)

	r, err := Load(repoPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !hasBadge(r.Worktrees[1], worktree.BadgeLocked) {
		t.Errorf("locked worktree should carry locked badge: %+v", r.Worktrees[1].Badges)
	}
}

func TestPopulateBadgesMissing(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "r")
	initRepo(t, repoPath)
	wtPath := filepath.Join(dir, "wt")
	addWorktree(t, repoPath, wtPath, "feat")
	if err := os.RemoveAll(wtPath); err != nil {
		t.Fatal(err)
	}

	r, err := Load(repoPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !hasBadge(r.Worktrees[1], worktree.BadgeMissing) {
		t.Errorf("removed worktree should carry missing badge: %+v", r.Worktrees[1].Badges)
	}
}

func TestDefaultBranchFallback(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	repoPath := filepath.Join(dir, "r")
	initRepo(t, repoPath)
	// No origin/HEAD configured, so the fallback to "main" should win.
	if got := defaultBranch(repoPath); got != "main" {
		t.Errorf("defaultBranch() = %q, want %q", got, "main")
	}
}

func hasBadge(w worktree.Worktree, b worktree.Badge) bool {
	return slices.Contains(w.Badges, b)
}
