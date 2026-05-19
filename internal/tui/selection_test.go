package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtm/internal/repo"
	"github.com/tnagatomi/wtm/internal/worktree"
)

func selectionTestModel(t *testing.T) Model {
	t.Helper()
	repos := []repo.Repo{{
		Path: "/repo",
		Worktrees: []worktree.Worktree{
			{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
			{Path: "/repo/wt/a", Branch: "feat-a"},
			{Path: "/repo/wt/b", Branch: "feat-b"},
		},
	}}
	m := tea.Model(NewModel(repos))
	// Window size first so the table viewport renders rows.
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	return m.(Model)
}

func TestSpaceTogglesSelectionOnSelectableRow(t *testing.T) {
	m := selectionTestModel(t)
	// Worktrees sort oldest-first by LastCommit; all three have zero time, so
	// they retain insertion order: primary at index 0. Move cursor to index 1.
	mAny := tea.Model(m)
	mAny, _ = mAny.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mAny, _ = mAny.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := mAny.(Model)
	if !got.selected[1] {
		t.Fatalf("space did not select index 1: %v", got.selected)
	}
	mAny, _ = mAny.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got = mAny.(Model)
	if got.selected[1] {
		t.Fatalf("second space did not deselect index 1: %v", got.selected)
	}
}

func TestSpaceIsNoOpOnPrimaryRow(t *testing.T) {
	m := selectionTestModel(t)
	mAny := tea.Model(m)
	// Cursor starts at index 0 (the primary worktree).
	mAny, _ = mAny.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := mAny.(Model)
	if len(got.selected) != 0 {
		t.Fatalf("space on primary row selected something: %v", got.selected)
	}
}

func TestCheckboxColumnRendersThreeStates(t *testing.T) {
	m := selectionTestModel(t)
	// Move to non-primary index 1 and select it; index 2 stays unselected;
	// index 0 is the non-selectable primary.
	mAny := tea.Model(m)
	mAny, _ = mAny.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	mAny, _ = mAny.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	view := mAny.(Model).View().Content

	if !strings.Contains(view, "[x]") {
		t.Errorf("view missing [x] for selected row: %q", view)
	}
	if !strings.Contains(view, "[ ]") {
		t.Errorf("view missing [ ] for unselected selectable row: %q", view)
	}
	// Primary row should render a blank cell — assert by counting [x]/[ ]
	// markers; exactly two rows are selectable, so we expect at most two
	// checkbox glyphs total.
	if strings.Count(view, "[x]")+strings.Count(view, "[ ]") != 2 {
		t.Errorf("expected 2 checkbox markers (one [x], one [ ]); got view %q", view)
	}
}

func TestSpaceNotPagingOnWorktreeTable(t *testing.T) {
	// Build a worktree list large enough that PageDown would visibly jump.
	wts := make([]worktree.Worktree, 20)
	for i := range wts {
		wts[i] = worktree.Worktree{Path: "/repo/wt", Branch: "b"}
	}
	repos := []repo.Repo{{Path: "/repo", Worktrees: wts}}
	m := tea.Model(NewModel(repos))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 8})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	// Cursor must not have paged — selection consumed space instead.
	if got := m.(Model).worktreeTable.Cursor(); got != 0 {
		t.Fatalf("space paged the cursor to %d; expected 0 (space toggles selection on worktree table)", got)
	}
}
