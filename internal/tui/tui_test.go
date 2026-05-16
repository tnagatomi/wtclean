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
