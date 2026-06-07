package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/worktree"
)

func filterTestModel(t *testing.T) tea.Model {
	return worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/Alpha", Branch: "release"},
		{Path: "/repo/wt/beta", Branch: "feat-foo"},
		{Path: "/repo/wt/gamma", Branch: "dirty",
			Badges: []worktree.Badge{worktree.BadgeDirty}},
	})
}

func typeFilter(m tea.Model, query string) tea.Model {
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range query {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return m
}

func TestFilterMatchesPathCaseInsensitive(t *testing.T) {
	m := filterTestModel(t)
	m = typeFilter(m, "ALPHA")
	got := m.(Model)
	if len(got.worktreeVisible) != 1 || got.worktreeVisible[0].Path != "/repo/wt/Alpha" {
		t.Fatalf("ALPHA query: visible = %v", got.worktreeVisible)
	}
}

func TestFilterMatchesBranchSubstring(t *testing.T) {
	m := filterTestModel(t)
	m = typeFilter(m, "foo")
	got := m.(Model)
	if len(got.worktreeVisible) != 1 || got.worktreeVisible[0].Branch != "feat-foo" {
		t.Fatalf("foo query: visible = %v", got.worktreeVisible)
	}
}

func TestFilterDoesNotMatchBadgeName(t *testing.T) {
	m := filterTestModel(t)
	m = typeFilter(m, "dirty")
	got := m.(Model)
	// The branch named "dirty" still matches by branch text; the [dirty] BADGE
	// on a different worktree should not be a separate match. Assert exactly
	// one row visible — the one whose branch is literally "dirty".
	if len(got.worktreeVisible) != 1 || got.worktreeVisible[0].Branch != "dirty" {
		t.Fatalf("dirty query should match only the branch named dirty: %v", got.worktreeVisible)
	}
}

func TestEscClearsFilterThenReturnsToScreen1(t *testing.T) {
	m := filterTestModel(t)
	m = typeFilter(m, "alpha")
	// First esc while in filter edit mode clears the query and exits edit.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	got := m.(Model)
	if got.filterEditing || got.filterQuery != "" {
		t.Fatalf("first esc should clear filter: editing=%v query=%q", got.filterEditing, got.filterQuery)
	}
	if got.screen != screenWorktrees {
		t.Fatalf("first esc should NOT leave Screen 2 when it just cleared the filter: screen=%v", got.screen)
	}
	// Second esc with empty query returns to Screen 1.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if got := m.(Model).screen; got != screenRepos {
		t.Fatalf("second esc should return to Screen 1: screen=%v", got)
	}
}

func TestEnterExitsFilterEditKeepingQuery(t *testing.T) {
	m := filterTestModel(t)
	m = typeFilter(m, "alpha")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := m.(Model)
	if got.filterEditing {
		t.Fatalf("enter should exit filter edit mode")
	}
	if got.filterQuery != "alpha" {
		t.Fatalf("enter should keep the query intact: %q", got.filterQuery)
	}
	if len(got.worktreeVisible) != 1 {
		t.Fatalf("filter should still apply after enter: visible = %v", got.worktreeVisible)
	}
}

func TestBackspaceShrinksFilterQuery(t *testing.T) {
	m := filterTestModel(t)
	m = typeFilter(m, "alpha")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyBackspace})
	got := m.(Model)
	if got.filterQuery != "alph" {
		t.Fatalf("backspace should remove last rune: %q", got.filterQuery)
	}
}

func TestQuitKeyIsLiteralWhileFiltering(t *testing.T) {
	m := filterTestModel(t)
	m = typeFilter(m, "q")
	got := m.(Model)
	if got.filterQuery != "q" {
		t.Fatalf("q while filtering should be a literal char, not quit: query=%q", got.filterQuery)
	}
}

func TestSelectionSurvivesFilterChange(t *testing.T) {
	m := filterTestModel(t)
	// Move cursor to /repo/wt/beta (index 1) and select it.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	if !m.(Model).selected["/repo/wt/beta"] {
		t.Fatalf("setup: beta should be selected")
	}
	// Filter to hide beta, then clear the filter.
	m = typeFilter(m, "alpha")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if !m.(Model).selected["/repo/wt/beta"] {
		t.Fatalf("beta should still be selected after filter cycle: %v", m.(Model).selected)
	}
}

func TestFilterQueryShownInView(t *testing.T) {
	m := filterTestModel(t)
	m = typeFilter(m, "alpha")
	view := m.(Model).View().Content
	if !strings.Contains(view, "/alpha") {
		t.Errorf("view should show the active filter query: %q", view)
	}
}
