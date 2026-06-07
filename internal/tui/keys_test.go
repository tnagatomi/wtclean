package tui

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/worktree"
)

func TestEmacsKeysMoveRepoTable(t *testing.T) {
	repos := []repo.Repo{
		{Path: "/r/a"}, {Path: "/r/b"}, {Path: "/r/c"}, {Path: "/r/d"},
	}
	m := tea.Model(NewModel(repos, ModelOptions{}))
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	if got := m.(Model).repoTable.Cursor(); got != 2 {
		t.Fatalf("after two ctrl+n: cursor = %d, want 2", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if got := m.(Model).repoTable.Cursor(); got != 1 {
		t.Fatalf("after ctrl+p: cursor = %d, want 1", got)
	}
}

func TestEmacsKeysMoveWorktreeTable(t *testing.T) {
	repos := []repo.Repo{{
		Path: "/repo",
		Worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "main"},
			{Path: "/repo/wt/a", Branch: "a"},
			{Path: "/repo/wt/b", Branch: "b"},
		},
	}}
	m := tea.Model(NewModel(repos, ModelOptions{}))
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Mod: tea.ModCtrl})
	if got := m.(Model).worktreeTable.Cursor(); got != 2 {
		t.Fatalf("after two ctrl+n on worktree screen: cursor = %d, want 2", got)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'p', Mod: tea.ModCtrl})
	if got := m.(Model).worktreeTable.Cursor(); got != 1 {
		t.Fatalf("after ctrl+p on worktree screen: cursor = %d, want 1", got)
	}
}

func TestEmacsPageKeysAdvanceMoreThanOneRow(t *testing.T) {
	repos := make([]repo.Repo, 30)
	for i := range repos {
		repos[i] = repo.Repo{Path: "/r"}
	}
	m := tea.Model(NewModel(repos, ModelOptions{}))
	// Constrain the viewport so page motion is well-defined; without a
	// non-zero height the table treats every move as a single row.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 10})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'v', Mod: tea.ModCtrl})
	pageDownCursor := m.(Model).repoTable.Cursor()
	if pageDownCursor < 2 {
		t.Fatalf("ctrl+v should page down by more than one row, cursor = %d", pageDownCursor)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: 'v', Mod: tea.ModAlt})
	if got := m.(Model).repoTable.Cursor(); got != 0 {
		t.Fatalf("alt+v should page back to the top, cursor = %d", got)
	}
}
