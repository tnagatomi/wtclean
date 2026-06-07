package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/worktree"
)

func TestQuestionTogglesHelpOverlay(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if !m.(Model).helpVisible {
		t.Fatal("? should open the help overlay")
	}
	view := m.(Model).View().Content
	if !strings.Contains(view, "keyboard reference") {
		t.Errorf("help view should render the reference header: %q", view)
	}
	if !strings.Contains(view, "toggle this help") {
		t.Errorf("help view should list the ? binding: %q", view)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if m.(Model).helpVisible {
		t.Fatal("second ? should close the help overlay")
	}
}

func TestEscClosesHelpOverlayOnly(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	// Open help, then esc should close it without leaving Screen 2.
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	got := m.(Model)
	if got.helpVisible {
		t.Error("esc should close the help overlay")
	}
	if got.screen != screenWorktrees {
		t.Errorf("esc while help was up should not navigate away: %v", got.screen)
	}
}

func TestKeysIgnoredWhileHelpVisible(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/a", Branch: "a"},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	// While help is up, "j" should not move the cursor.
	cursorBefore := m.(Model).worktreeTable.Cursor()
	m, _ = m.Update(tea.KeyPressMsg{Code: 'j', Text: "j"})
	if got := m.(Model).worktreeTable.Cursor(); got != cursorBefore {
		t.Fatalf("cursor should not move while help is up: %d -> %d", cursorBefore, got)
	}
}

func TestQuestionInFilterEditingIsLiteral(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	// Enter filter mode then type "?". It should be appended to the
	// query rather than opening the help overlay (filter-edit takes
	// precedence over global hotkeys).
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	got := m.(Model)
	if got.helpVisible {
		t.Error("? in filter-edit mode should not open help")
	}
	if got.filterQuery != "?" {
		t.Errorf("? in filter-edit mode should be appended to the query: %q", got.filterQuery)
	}
}
