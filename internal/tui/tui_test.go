package tui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtm/internal/repo"
	"github.com/tnagatomi/wtm/internal/worktree"
)

func TestViewEmpty(t *testing.T) {
	m := NewModel(nil)
	view := m.View().Content
	if !strings.Contains(view, "No repositories") {
		t.Errorf("empty view missing message: %q", view)
	}
}

func TestViewListsRepos(t *testing.T) {
	repos := []repo.Repo{
		{
			Path:      "/home/u/alpha",
			Worktrees: make([]worktree.Worktree, 3),
			LastFetch: time.Date(2026, 5, 17, 10, 30, 0, 0, time.UTC),
		},
		{
			Path:      "/home/u/beta",
			Worktrees: make([]worktree.Worktree, 2),
		},
	}
	m := NewModel(repos)
	view := m.View().Content
	if !strings.Contains(view, "/home/u/alpha") {
		t.Errorf("view missing alpha repo: %q", view)
	}
	if !strings.Contains(view, "/home/u/beta") {
		t.Errorf("view missing beta repo: %q", view)
	}
	if !strings.Contains(view, "2026-05-17 10:30:00") {
		t.Errorf("view missing formatted fetch time: %q", view)
	}
	if !strings.Contains(view, emptyTime) {
		t.Errorf("view missing %q placeholder for zero fetch time: %q", emptyTime, view)
	}
}

func TestFormatTime(t *testing.T) {
	if got := formatTime(time.Time{}); got != emptyTime {
		t.Errorf("zero time: got %q, want %q", got, emptyTime)
	}
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := formatTime(at); got != "2026-01-02 03:04:05" {
		t.Errorf("got %q", got)
	}
}

func TestSortWorktreesOldestFirst(t *testing.T) {
	wts := []worktree.Worktree{
		{Path: "/recent", LastCommit: time.Date(2026, 5, 17, 0, 0, 0, 0, time.UTC)},
		{Path: "/zero"},
		{Path: "/old", LastCommit: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)},
		{Path: "/middle", LastCommit: time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)},
	}
	got := sortWorktrees(wts)
	want := []string{"/zero", "/old", "/middle", "/recent"}
	for i, w := range got {
		if w.Path != want[i] {
			t.Errorf("[%d] got %s, want %s", i, w.Path, want[i])
		}
	}
}

func TestEnterNavigatesToWorktreeScreen(t *testing.T) {
	repos := []repo.Repo{{
		Path: "/repo-a",
		Worktrees: []worktree.Worktree{
			{Path: "/repo-a", Branch: "main", LastCommit: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
			{Path: "/repo-a/wt/feat", Branch: "feat-x", LastCommit: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)},
		},
	}}
	m := tea.Model(NewModel(repos))
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got := m.(Model)
	if got.screen != screenWorktrees {
		t.Fatalf("screen: got %v, want screenWorktrees", got.screen)
	}
	if got.selectedRepoIdx != 0 {
		t.Errorf("selectedRepoIdx: got %d, want 0", got.selectedRepoIdx)
	}
	view := got.View().Content
	if !strings.Contains(view, "/repo-a/wt/feat") {
		t.Errorf("view missing worktree path: %q", view)
	}
	if !strings.Contains(view, "feat-x") {
		t.Errorf("view missing branch: %q", view)
	}
	if !strings.Contains(view, "2026-01-01") {
		t.Errorf("view missing last commit date: %q", view)
	}
}

func TestEscReturnsToRepoScreen(t *testing.T) {
	repos := []repo.Repo{{
		Path: "/repo-a",
		Worktrees: []worktree.Worktree{
			{Path: "/repo-a", Branch: "main"},
			{Path: "/repo-a/wt/feat", Branch: "feat-x"},
		},
	}}
	m := tea.Model(NewModel(repos))
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	if got := m.(Model).screen; got != screenRepos {
		t.Errorf("screen after esc: got %v, want screenRepos", got)
	}
}

func TestEnterIgnoredWhenNoRepos(t *testing.T) {
	m := tea.Model(NewModel(nil))
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if got := m.(Model).screen; got != screenRepos {
		t.Errorf("screen with no repos: got %v, want screenRepos", got)
	}
}

func TestTruncateHead(t *testing.T) {
	cases := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"fits exactly", "/a/b/c", 6, "/a/b/c"},
		{"under width", "/a/b/c", 10, "/a/b/c"},
		{"truncates head with ellipsis", "/a/b/c", 5, ellipsis + "/b/c"},
		{"narrow leaves only ellipsis and tail", "/a/b/c", 3, ellipsis + "/c"},
		{"width 2 leaves ellipsis plus one rune", "/abcdef", 2, ellipsis + "f"},
		{"width 1 returns ellipsis", "/a/b/c", 1, ellipsis},
		{"width 0 returns empty", "/a/b/c", 0, ""},
		{"empty input returns empty regardless of width", "", 5, ""},
		{"multi-rune input is sliced by rune not byte", "/日本語/repo", 6, ellipsis + "/repo"},
		{"preserves repository name at tail", "/Users/u/ghq/github.com/o/repo", 12, ellipsis + ".com/o/repo"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := truncateHead(c.in, c.width); got != c.want {
				t.Errorf("truncateHead(%q, %d) = %q, want %q", c.in, c.width, got, c.want)
			}
		})
	}
}
