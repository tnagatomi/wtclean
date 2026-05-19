package tui

import (
	"strings"
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
	m := tea.Model(NewModel(repos, ModelOptions{}))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return m
}

// lineContaining returns the first line of view that contains needle, or
// "" if none match. Used by view tests to assert per-line content without
// hard-coding line offsets that shift as the layout evolves.
func lineContaining(view, needle string) string {
	for _, line := range strings.Split(view, "\n") {
		if strings.Contains(line, needle) {
			return line
		}
	}
	return ""
}
