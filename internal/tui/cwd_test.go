package tui

import (
	"strings"
	"testing"

	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/worktree"
)

func cwdRepo() repo.Repo {
	return repo.Repo{
		Path: "/repo-a",
		Worktrees: []worktree.Worktree{
			{Path: "/repo-a", Branch: "main"},
			{Path: "/repo-a/wt/feat", Branch: "feat-x"},
		},
	}
}

func TestNewSingleRepoOpensWorktreeScreen(t *testing.T) {
	m := NewSingleRepo(cwdRepo(), ModelOptions{})
	if m.screen != screenWorktrees {
		t.Fatalf("screen: got %v, want screenWorktrees", m.screen)
	}
	view := m.View().Content
	if !strings.Contains(view, "/repo-a/wt/feat") {
		t.Errorf("view missing worktree path: %q", view)
	}
	if !strings.Contains(view, "feat-x") {
		t.Errorf("view missing branch: %q", view)
	}
}

func TestNewSingleRepoInitDispatchesNoScan(t *testing.T) {
	m := NewSingleRepo(cwdRepo(), ModelOptions{})
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init should dispatch no scan in cwd mode, got a non-nil cmd")
	}
}
