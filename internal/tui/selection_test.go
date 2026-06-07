package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/worktree"
)

func selectionTestModel(t *testing.T) tea.Model {
	return worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/a", Branch: "feat-a"},
		{Path: "/repo/wt/b", Branch: "feat-b"},
	})
}

func TestSpaceTogglesSelectionOnSelectableRow(t *testing.T) {
	m := selectionTestModel(t)
	// Worktrees sort oldest-first by LastCommit; all three have zero time, so
	// they retain insertion order: primary at index 0. Move cursor to index 1.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if !m.(Model).selected["/repo/wt/a"] {
		t.Fatalf("space did not select /repo/wt/a: %v", m.(Model).selected)
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if m.(Model).selected["/repo/wt/a"] {
		t.Fatalf("second space did not deselect /repo/wt/a: %v", m.(Model).selected)
	}
}

func TestSpaceIsNoOpOnPrimaryRow(t *testing.T) {
	m := selectionTestModel(t)
	// Cursor starts at index 0 (the primary worktree).
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if got := m.(Model).selected; len(got) != 0 {
		t.Fatalf("space on primary row selected something: %v", got)
	}
}

func TestCheckboxColumnRendersThreeStates(t *testing.T) {
	m := selectionTestModel(t)
	// Move to non-primary index 1 and select it; index 2 stays unselected;
	// index 0 is the non-selectable primary.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	view := m.(Model).View().Content

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
	m := tea.Model(NewModel(repos, ModelOptions{}))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 8})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	// Cursor must not have paged — selection consumed space instead.
	if got := m.(Model).worktreeTable.Cursor(); got != 0 {
		t.Fatalf("space paged the cursor to %d; expected 0 (space toggles selection on worktree table)", got)
	}
}
