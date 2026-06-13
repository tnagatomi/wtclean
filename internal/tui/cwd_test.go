package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/worktree"
)

func cwdRepo() repo.Repo {
	return repo.Repo{
		Path: "/repo-a",
		Worktrees: []worktree.Worktree{
			{Path: "/repo-a", Branch: "main"},
			{Path: "/repo-a/wt/feat", Branch: "feat-x"},
		},
	}
}

func TestNewSingleRepoOpensWorktreeScreen(t *testing.T) {
	m := NewSingleRepo(cwdRepo(), ModelOptions{})
	if m.screen != screenWorktrees {
		t.Fatalf("screen: got %v, want screenWorktrees", m.screen)
	}
	view := m.View().Content
	if !strings.Contains(view, "/repo-a/wt/feat") {
		t.Errorf("view missing worktree path: %q", view)
	}
	if !strings.Contains(view, "feat-x") {
		t.Errorf("view missing branch: %q", view)
	}
}

func TestNewSingleRepoInitDispatchesNoScan(t *testing.T) {
	m := NewSingleRepo(cwdRepo(), ModelOptions{})
	if cmd := m.Init(); cmd != nil {
		t.Errorf("Init should dispatch no scan in cwd mode, got a non-nil cmd")
	}
}

func TestEscQuitsInCwdMode(t *testing.T) {
	m := tea.Model(NewSingleRepo(cwdRepo(), ModelOptions{}))
	got, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd == nil {
		t.Fatalf("esc in cwd mode should produce a quit command, got nil")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("esc in cwd mode should quit, got %T", cmd())
	}
	// The screen does not change to a (nonexistent) repository list.
	if got.(Model).screen != screenWorktrees {
		t.Errorf("screen after esc: got %v, want screenWorktrees", got.(Model).screen)
	}
}

func TestFooterWordingInCwdMode(t *testing.T) {
	m := NewSingleRepo(cwdRepo(), ModelOptions{})
	view := m.View().Content
	if strings.Contains(view, "[esc] back/clear") {
		t.Errorf("cwd-mode footer should not offer 'back', there is no screen to go back to: %q", view)
	}
	if !strings.Contains(view, "[esc] clear/quit") {
		t.Errorf("cwd-mode footer should describe esc as clear/quit: %q", view)
	}
}

func TestHelpEscWordingInCwdMode(t *testing.T) {
	m := tea.Model(NewSingleRepo(cwdRepo(), ModelOptions{}))
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"}) // open help overlay
	view := m.(Model).View().Content
	if strings.Contains(view, "back to Screen 1") {
		t.Errorf("cwd-mode help should not reference Screen 1: %q", view)
	}
	if !strings.Contains(view, "clear filter, or quit") {
		t.Errorf("cwd-mode help should describe esc as clear filter, or quit: %q", view)
	}
}

func TestEscClearsFilterBeforeQuittingInCwdMode(t *testing.T) {
	m := tea.Model(NewSingleRepo(cwdRepo(), ModelOptions{}))
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	for _, r := range "feat" {
		m, _ = m.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // commit the filter

	// esc with an active filter clears it rather than quitting.
	m, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if cmd != nil {
		t.Errorf("esc with an active filter should clear, not quit: got cmd %T", cmd())
	}
	if got := m.(Model); got.filterQuery != "" {
		t.Errorf("esc should clear the filter, got query %q", got.filterQuery)
	}
}
