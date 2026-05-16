package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/tnagatomi/wtm/internal/repo"
	"github.com/tnagatomi/wtm/internal/worktree"
)

func TestViewEmpty(t *testing.T) {
	m := NewModel(nil)
	view := m.View()
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
	view := m.View()
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
