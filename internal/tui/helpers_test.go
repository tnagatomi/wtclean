package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtm/internal/repo"
	"github.com/tnagatomi/wtm/internal/worktree"
)

// worktreeScreenModel returns a Model parked on Screen 2 for a single
// repo containing wts, with a window size set so the table viewport
// renders rows.
func worktreeScreenModel(t *testing.T, wts []worktree.Worktree) tea.Model {
	t.Helper()
	repos := []repo.Repo{{Path: "/repo", Worktrees: wts}}
	m := tea.Model(NewModel(repos))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return m
}
