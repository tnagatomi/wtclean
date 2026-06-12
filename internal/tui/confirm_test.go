package tui

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/deleter"
	"github.com/tnagatomi/wtclean/internal/worktree"
)

// confirmScreenModel returns a Model parked on the confirm screen with
// two selected non-primary worktrees (one merged, one dirty) and the
// primary worktree present but unselected.
func confirmScreenModel(t *testing.T) tea.Model {
	t.Helper()
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/feat", Branch: "feat", Badges: []worktree.Badge{worktree.BadgeMerged}},
		{Path: "/repo/wt/wip", Branch: "wip", Badges: []worktree.Badge{worktree.BadgeUncommitted}},
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
	if !strings.Contains(view, "uncommitted:") || !strings.Contains(view, "(1)") {
		t.Errorf("view missing uncommitted warning count: %q", view)
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

func TestYDispatchesDeleteCompleteMsg(t *testing.T) {
	m := confirmScreenModel(t)
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Fatal("y should dispatch a delete cmd; got nil")
	}
	// Run the cmd to pin down the dispatch contract; the real git calls
	// inside Delete will fail against the fake paths in the fixture, but
	// the Cmd is contractually required to return a deleteCompleteMsg
	// regardless of the underlying success.
	if _, ok := cmd().(deleteCompleteMsg); !ok {
		t.Fatalf("delete cmd should return deleteCompleteMsg; got %T", cmd())
	}
}

func TestYShowsDeletingIndicator(t *testing.T) {
	m := confirmScreenModel(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	got := m.(Model)
	if !got.deleting {
		t.Fatal("y should mark the model as deleting")
	}
	view := got.View().Content
	if !strings.Contains(view, "Deleting...") {
		t.Errorf("confirm view should show the deleting indicator: %q", view)
	}
	if strings.Contains(view, "[y] Confirm") {
		t.Errorf("confirm view should hide the action keys while deleting: %q", view)
	}
}

func TestKeysIgnoredWhileDeleting(t *testing.T) {
	m := confirmScreenModel(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	// A second confirm/cancel keypress must not change state while the
	// deletion is still in flight.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'n', Text: "n"})
	got := m.(Model)
	if got.screen != screenConfirmDelete {
		t.Fatalf("n should be inert while deleting: screen=%v", got.screen)
	}
	if !got.deleting {
		t.Fatal("model should remain in the deleting state")
	}
}

func TestDeleteCompleteClearsDeletingFlag(t *testing.T) {
	m := confirmScreenModel(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	m, _ = m.Update(deleteCompleteMsg{})
	if m.(Model).deleting {
		t.Fatal("deleting flag should be cleared once the deletion completes")
	}
}

func TestDeleteCompleteReturnsToScreen2WithFailures(t *testing.T) {
	m := confirmScreenModel(t)
	// Bypass the real git Cmd by synthesizing the completion message
	// directly. The deleter package owns the actual git execution tests.
	failures := []deleter.Failure{{Path: "/repo/wt/wip", Op: deleter.OpRemove}}
	m, _ = m.Update(deleteCompleteMsg{failures: failures})
	got := m.(Model)
	if got.screen != screenWorktrees {
		t.Fatalf("after delete completion screen should be Screen 2: %v", got.screen)
	}
	if len(got.selected) != 0 {
		t.Errorf("selection should be cleared after delete: %v", got.selected)
	}
	if len(got.deleteFailures) != 1 {
		t.Errorf("failures should be stashed on the model: %v", got.deleteFailures)
	}
}

func TestDeleteFailuresAreAppendedToLog(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	m := confirmScreenModel(t)
	failures := []deleter.Failure{
		{Path: "/repo/wt/wip", Op: deleter.OpRemove, Err: errors.New("boom")},
		{Path: "/repo/wt/old", Op: deleter.OpUnlock, Err: errors.New("locked by another process")},
	}
	_, cmd := m.Update(deleteCompleteMsg{failures: failures})
	if cmd == nil {
		t.Fatal("delete completion with failures should return a log cmd")
	}
	cmd() // execute the log cmd inline

	data, err := os.ReadFile(filepath.Join(dir, "wtclean", "wtclean.log"))
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	body := string(data)
	if !strings.Contains(body, "delete:remove /repo/wt/wip: boom") {
		t.Errorf("log missing remove failure: %q", body)
	}
	if !strings.Contains(body, "delete:unlock /repo/wt/old: locked by another process") {
		t.Errorf("log missing unlock failure: %q", body)
	}
}

func TestDeleteFailureSummaryRenderedOnScreen2(t *testing.T) {
	m := confirmScreenModel(t)
	failures := []deleter.Failure{
		{Path: "/repo/wt/a", Op: deleter.OpRemove},
		{Path: "/repo/wt/b", Op: deleter.OpRemove},
	}
	m, _ = m.Update(deleteCompleteMsg{failures: failures})
	view := m.(Model).View().Content
	if !strings.Contains(view, "2 operation(s) failed during last delete") {
		t.Errorf("Screen 2 should surface failure count: %q", view)
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
