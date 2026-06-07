package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/worktree"
)

func pressSafeSelect(m tea.Model) tea.Model {
	m, _ = m.Update(tea.KeyPressMsg{Code: 's', Text: "s"})
	return m
}

func TestSafeSelectSelectsMergedWorktree(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/merged", Branch: "merged", Badges: []worktree.Badge{worktree.BadgeMerged}},
		{Path: "/repo/wt/wip", Branch: "wip", Badges: []worktree.Badge{worktree.BadgeUncommitted}},
	})
	m = pressSafeSelect(m)
	if !m.(Model).selected["/repo/wt/merged"] {
		t.Fatalf("s did not select the merged worktree: %v", m.(Model).selected)
	}
}

// T2: only safe-to-remove rows are selected. A warning row (uncommitted), a
// clean-but-unmerged active row (no badges), and the primary — even when it
// carries [merged] — must all be left untouched.
func TestSafeSelectIgnoresNonSafeRows(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary, worktree.BadgeMerged}},
		{Path: "/repo/wt/merged", Branch: "merged", Badges: []worktree.Badge{worktree.BadgeMerged}},
		{Path: "/repo/wt/wip", Branch: "wip", Badges: []worktree.Badge{worktree.BadgeUncommitted}},
		{Path: "/repo/wt/active", Branch: "active"},
	})
	m = pressSafeSelect(m)
	got := m.(Model).selected
	if len(got) != 1 || !got["/repo/wt/merged"] {
		t.Fatalf("expected only the merged worktree selected, got %v", got)
	}
}

// T3: pressing s again, when every visible safe row is already selected,
// deselects them (stateless toggle).
func TestSafeSelectTogglesOffWhenAllSafeSelected(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/a", Branch: "a", Badges: []worktree.Badge{worktree.BadgeMerged}},
		{Path: "/repo/wt/b", Branch: "b", Badges: []worktree.Badge{worktree.BadgeUpstreamGone}},
	})
	m = pressSafeSelect(m)
	if len(m.(Model).selected) != 2 {
		t.Fatalf("first s should select both safe rows: %v", m.(Model).selected)
	}
	m = pressSafeSelect(m)
	if len(m.(Model).selected) != 0 {
		t.Fatalf("second s should deselect the safe rows: %v", m.(Model).selected)
	}
}

// T4: a manually selected non-safe row survives a press of s — the toggle only
// touches the safe set. Pressing s off again leaves the manual selection intact.
func TestSafeSelectPreservesManualNonSafeSelection(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/active", Branch: "active"},
		{Path: "/repo/wt/merged", Branch: "merged", Badges: []worktree.Badge{worktree.BadgeMerged}},
	})
	// Manually select the active (non-safe) row at index 1.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m = pressSafeSelect(m)
	if !m.(Model).selected["/repo/wt/active"] || !m.(Model).selected["/repo/wt/merged"] {
		t.Fatalf("s should add safe without dropping the manual selection: %v", m.(Model).selected)
	}
	m = pressSafeSelect(m)
	got := m.(Model).selected
	if got["/repo/wt/merged"] {
		t.Fatalf("second s should deselect the safe row: %v", got)
	}
	if !got["/repo/wt/active"] {
		t.Fatalf("manual non-safe selection must survive the toggle: %v", got)
	}
}

// T5: under an active filter, s only affects the visible safe rows; safe rows
// hidden by the filter are left unselected.
func TestSafeSelectScopedToVisibleUnderFilter(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/keep", Branch: "keep", Badges: []worktree.Badge{worktree.BadgeMerged}},
		{Path: "/repo/wt/hide", Branch: "hide", Badges: []worktree.Badge{worktree.BadgeMerged}},
	})
	m = typeFilter(m, "keep")
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // apply filter, leave edit mode
	m = pressSafeSelect(m)
	got := m.(Model).selected
	if !got["/repo/wt/keep"] {
		t.Fatalf("visible safe row should be selected: %v", got)
	}
	if got["/repo/wt/hide"] {
		t.Fatalf("safe row hidden by the filter must not be selected: %v", got)
	}
}

// T6: with no safe rows visible, s is a no-op.
func TestSafeSelectNoOpWhenNoSafeRows(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/wip", Branch: "wip", Badges: []worktree.Badge{worktree.BadgeUncommitted}},
		{Path: "/repo/wt/active", Branch: "active"},
	})
	m = pressSafeSelect(m)
	if got := m.(Model).selected; len(got) != 0 {
		t.Fatalf("s with no safe rows should select nothing: %v", got)
	}
}

// T7a: the Screen 2 footer advertises the s binding.
func TestWorktreeFooterShowsSafeSelectKey(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/a", Branch: "a", Badges: []worktree.Badge{worktree.BadgeMerged}},
	})
	view := m.(Model).View().Content
	if !strings.Contains(view, "[s]") {
		t.Errorf("worktree footer should advertise the [s] safe-select key: %q", view)
	}
}

// T7b: the help overlay documents the s binding and the badge legend, grouped
// into safe-to-remove vs warning states.
func TestHelpOverlayListsSafeSelectAndBadgeLegend(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	view := m.(Model).View().Content
	for _, want := range []string{
		"select all safe-to-remove worktrees",
		"Badges — safe to remove",
		"[upstream-gone]",
		"[no-dir]",
		"Badges — removal loses local work",
		"[uncommitted]",
	} {
		if !strings.Contains(view, want) {
			t.Errorf("help overlay missing %q:\n%s", want, view)
		}
	}
}
