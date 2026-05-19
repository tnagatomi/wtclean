package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtm/internal/worktree"
)

// confirmScreenModel returns a Model parked on the confirm screen with
// two selected non-primary worktrees (one merged, one dirty) and the
// primary worktree present but unselected.
func confirmScreenModel(t *testing.T) tea.Model {
	t.Helper()
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/feat", Branch: "feat", Badges: []worktree.Badge{worktree.BadgeMerged}},
		{Path: "/repo/wt/wip", Branch: "wip", Badges: []worktree.Badge{worktree.BadgeDirty}},
	})
	// Select both non-primary rows.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	return m
}

func TestDKeyWithNoSelectionStaysOnScreen2(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/a", Branch: "a"},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if got := m.(Model).screen; got != screenWorktrees {
		t.Fatalf("d with no selection should stay on Screen 2: screen=%v", got)
	}
}

func TestDKeyWithSelectionEntersConfirmScreen(t *testing.T) {
	m := confirmScreenModel(t)
	got := m.(Model)
	if got.screen != screenConfirmDelete {
		t.Fatalf("d with selection should enter confirm screen: screen=%v", got.screen)
	}
	if len(got.deleteTargets) != 2 {
		t.Fatalf("deleteTargets: got %d, want 2", len(got.deleteTargets))
	}
}

func TestConfirmViewListsTargets(t *testing.T) {
	m := confirmScreenModel(t)
	view := m.(Model).View().Content
	if !strings.Contains(view, "Deleting 2 worktrees:") {
		t.Errorf("view missing header: %q", view)
	}
	if !strings.Contains(view, "/repo/wt/feat") || !strings.Contains(view, "/repo/wt/wip") {
		t.Errorf("view missing target paths: %q", view)
	}
	if strings.Contains(view, "/repo/wt/main") {
		t.Errorf("view should not list unselected primary: %q", view)
	}
}

func TestConfirmViewMarksWarningRow(t *testing.T) {
	m := confirmScreenModel(t)
	view := m.(Model).View().Content
	// The dirty row should be prefixed with ⚠; the merged-only row should not.
	dirtyLine := lineContaining(view, "/repo/wt/wip")
	if !strings.Contains(dirtyLine, "⚠") {
		t.Errorf("dirty target should be marked with ⚠: %q", dirtyLine)
	}
	featLine := lineContaining(view, "/repo/wt/feat")
	if strings.Contains(featLine, "⚠") {
		t.Errorf("merged-only target should not be marked with ⚠: %q", featLine)
	}
}

func TestConfirmViewShowsWarningCounts(t *testing.T) {
	m := confirmScreenModel(t)
	view := m.(Model).View().Content
	if !strings.Contains(view, "⚠ Warnings (deletion will be forced):") {
		t.Errorf("view missing warnings block header: %q", view)
	}
	if !strings.Contains(view, "dirty:") || !strings.Contains(view, "(1)") {
		t.Errorf("view missing dirty warning count: %q", view)
	}
}

func TestBranchToggleDefaultOnWhenAllMerged(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/a", Branch: "a", Badges: []worktree.Badge{worktree.BadgeMerged}},
		{Path: "/repo/wt/b", Branch: "b", Badges: []worktree.Badge{worktree.BadgeMerged}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if !m.(Model).deleteBranchesToggle {
		t.Fatalf("toggle should default ON when every selected is merged")
	}
}

func TestBranchToggleDefaultOffOtherwise(t *testing.T) {
	if got := confirmScreenModel(t).(Model).deleteBranchesToggle; got {
		t.Fatalf("toggle should default OFF when any selected is not merged: got %v", got)
	}
}

func TestSpaceTogglesBranchOption(t *testing.T) {
	m := confirmScreenModel(t)
	initial := m.(Model).deleteBranchesToggle
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if m.(Model).deleteBranchesToggle == initial {
		t.Fatalf("space should flip the branches toggle")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if m.(Model).deleteBranchesToggle != initial {
		t.Fatalf("second space should flip the toggle back")
	}
}

func TestNCancelsBackToScreen2(t *testing.T) {
	m := confirmScreenModel(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	if got := m.(Model).screen; got != screenWorktrees {
		t.Fatalf("n should return to Screen 2: screen=%v", got)
	}
}

func TestEscCancelsBackToScreen2(t *testing.T) {
	m := confirmScreenModel(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if got := m.(Model).screen; got != screenWorktrees {
		t.Fatalf("esc should return to Screen 2: screen=%v", got)
	}
}

func TestDKeyDoesNotHalfPageDownOnWorktreeTable(t *testing.T) {
	// Build a list big enough that a HalfPageDown would visibly move the
	// cursor, but without any selection so `d` is a no-op (rather than
	// entering the confirm screen).
	wts := make([]worktree.Worktree, 20)
	wts[0] = worktree.Worktree{Path: "/repo", Badges: []worktree.Badge{worktree.BadgePrimary}}
	for i := 1; i < len(wts); i++ {
		wts[i] = worktree.Worktree{Path: "/repo/wt", Branch: "b"}
	}
	m := worktreeScreenModel(t, wts)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if got := m.(Model).worktreeTable.Cursor(); got != 0 {
		t.Fatalf("d should not page the cursor on the worktree table; cursor=%d", got)
	}
}

