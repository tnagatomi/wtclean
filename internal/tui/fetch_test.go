package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/worktree"
)

func TestRKeyDispatchesFetchCmdAndMarksFetching(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("r should dispatch a fetch cmd")
	}
	got := updated.(Model)
	if !got.fetching {
		t.Error("r should mark Model.fetching = true so the view can show a spinner/indicator")
	}
}

func TestRWhileFetchingIsNoOp(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	// First r kicks off a fetch.
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	// Second r while still fetching should not start another one.
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd != nil {
		t.Fatal("second r while fetching should be a no-op (got non-nil cmd)")
	}
}

func TestFetchCompleteClearsFetchingAndUpdatesRepo(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	// Simulate a successful completion with a freshly-loaded repo.
	fresh := repo.Repo{Path: "/repo", Worktrees: []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/new", Branch: "new"},
	}}
	m, _ = m.Update(fetchCompleteMsg{reloaded: &fresh})
	got := m.(Model)
	if got.fetching {
		t.Error("fetching should be cleared after fetchCompleteMsg")
	}
	if got.fetchError != nil {
		t.Errorf("fetchError should be nil on success: %v", got.fetchError)
	}
	if len(got.repos[got.selectedRepoIdx].Worktrees) != 2 {
		t.Errorf("repo should be replaced with fresh data; got %d worktrees", len(got.repos[got.selectedRepoIdx].Worktrees))
	}
}

func TestFetchErrorSurfacedInView(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	m, _ = m.Update(fetchCompleteMsg{fetchErr: errors.New("network down")})
	view := m.(Model).View().Content
	if !strings.Contains(view, "fetch failed") || !strings.Contains(view, "network down") {
		t.Errorf("view should surface fetch error: %q", view)
	}
}

func TestFetchCmdSequencesFetchThenLoadAndReturnsTypedMsg(t *testing.T) {
	// Point the model at a nonexistent path so both fetcher.Fetch and
	// repo.Load fail — this lets us assert the Cmd actually wired both
	// calls AND packed both error fields, not just one.
	model := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	}).(Model)
	model.repos[model.selectedRepoIdx].Path = "/nonexistent/path/for/fetch/cmd/test"

	cmd := model.fetchCmd()
	if cmd == nil {
		t.Fatal("fetchCmd should return a non-nil tea.Cmd")
	}
	msg, ok := cmd().(fetchCompleteMsg)
	if !ok {
		t.Fatalf("cmd should return a fetchCompleteMsg, got %T", cmd())
	}
	if msg.fetchErr == nil {
		t.Error("fetchCmd should populate fetchErr when git fetch fails (you forgot to call fetcher.Fetch?)")
	}
	if msg.loadErr == nil {
		t.Error("fetchCmd should populate loadErr when repo.Load fails (you forgot to call repo.Load?)")
	}
	if msg.reloaded != nil {
		t.Errorf("reloaded should be nil when repo.Load failed, got %+v", msg.reloaded)
	}
}

func TestFetchFailurePreservesSelectionAndFilter(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Path: "/repo/wt/a", Branch: "feat-a"},
	})
	// Establish some user context: select a row and start a filter.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	m, _ = m.Update(tea.KeyPressMsg{Code: '/', Text: "/"})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'f', Text: "f"})
	// Fetch fails — load also fails (no fresh data).
	m, _ = m.Update(fetchCompleteMsg{fetchErr: errors.New("network down")})
	got := m.(Model)
	if !got.selected["/repo/wt/a"] {
		t.Errorf("selection should survive a failed fetch: %v", got.selected)
	}
	if got.filterQuery != "f" {
		t.Errorf("filter query should survive a failed fetch: %q", got.filterQuery)
	}
}

func TestFetchingIndicatorRendered(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	view := m.(Model).View().Content
	if !strings.Contains(view, "Fetching") {
		t.Errorf("view should show a fetching indicator while m.fetching is true: %q", view)
	}
}
