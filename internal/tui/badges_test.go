package tui

import (
	"strings"
	"testing"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2/compat"
	"github.com/charmbracelet/x/ansi"

	"github.com/tnagatomi/wtm/internal/worktree"
)

func TestRenderBadgesEmpty(t *testing.T) {
	if got := renderBadges(nil); got != "" {
		t.Errorf("empty badges: got %q, want empty string", got)
	}
}

func TestRenderBadgesContainsLabels(t *testing.T) {
	rendered := renderBadges([]worktree.Badge{
		worktree.BadgePrimary,
		worktree.BadgeMerged,
		worktree.BadgeDirty,
	})
	for _, want := range []string{"[primary]", "[merged]", "[dirty]"} {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered badges missing %q: %q", want, rendered)
		}
	}
}

func TestPlainBadgesWidth(t *testing.T) {
	cases := []struct {
		name   string
		badges []worktree.Badge
		want   int
	}{
		{"empty", nil, 0},
		{"primary alone is 9", []worktree.Badge{worktree.BadgePrimary}, 9},
		{"merged alone is 8", []worktree.Badge{worktree.BadgeMerged}, 8},
		{"dirty alone is 7", []worktree.Badge{worktree.BadgeDirty}, 7},
		{"primary plus merged is 18", []worktree.Badge{worktree.BadgePrimary, worktree.BadgeMerged}, 18},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := plainBadgesWidth(c.badges); got != c.want {
				t.Errorf("plainBadgesWidth(%v) = %d, want %d", c.badges, got, c.want)
			}
		})
	}
}

func TestBadgesVisibleWidthMatchesLongest(t *testing.T) {
	wts := []worktree.Worktree{
		{Badges: []worktree.Badge{worktree.BadgePrimary}},
		{Badges: []worktree.Badge{worktree.BadgeMerged, worktree.BadgeDirty}},
		{Badges: nil},
	}
	// "[merged] [dirty]" has visible width 8 + 1 + 7 = 16.
	if got := badgesVisibleWidth(wts); got != 16 {
		t.Errorf("badgesVisibleWidth = %d, want 16", got)
	}
}

func TestBadgesVisibleWidthHasHeaderFloor(t *testing.T) {
	if got := badgesVisibleWidth(nil); got != len("Badges") {
		t.Errorf("badgesVisibleWidth(nil) = %d, want %d", got, len("Badges"))
	}
}

func TestRenderBadgesNotTruncatedWithColorProfile(t *testing.T) {
	withDarkBackground(t)

	wts := []worktree.Worktree{
		{Path: "/repo", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary}},
	}
	cols, rows := worktreeLayout(wts, nil, len("Path"), len("Branch"), badgesVisibleWidth(wts), 200)
	tb := table.New(table.WithColumns(cols), table.WithRows(rows), table.WithWidth(tableWidth(cols)))
	view := ansi.Strip(tb.View())

	if !strings.Contains(view, "[primary]") {
		t.Fatalf("view missing full badge: %q", view)
	}
	if strings.Contains(view, "[primar"+ellipsis) {
		t.Fatalf("view contains truncated badge: %q", view)
	}
}

// withDarkBackground forces compat.HasDarkBackground = true for the duration
// of the test so AdaptiveColor resolves to the Dark variant deterministically
// regardless of the host terminal. lipgloss v2 styles always emit ANSI on
// Render, so no separate color-profile override is needed.
func withDarkBackground(t *testing.T) {
	t.Helper()
	old := compat.HasDarkBackground
	compat.HasDarkBackground = true
	t.Cleanup(func() { compat.HasDarkBackground = old })
}

func TestRenderWorktreeTableColorsRowsByBadge(t *testing.T) {
	withDarkBackground(t)

	wts := []worktree.Worktree{
		{Path: "/repo", Branch: "main"},
		{Path: "/repo/wt/dirty", Branch: "dirty", Badges: []worktree.Badge{worktree.BadgeDirty}},
		{Path: "/repo/wt/primary", Branch: "main", Badges: []worktree.Badge{worktree.BadgePrimary, worktree.BadgeMerged}},
	}
	pathWidth, branchWidth := maxWorktreeWidths(wts)
	badgesWidth := badgesVisibleWidth(wts)
	cols, rows := worktreeLayout(wts, nil, pathWidth, branchWidth, badgesWidth, 200)
	tb := table.New(
		table.WithColumns(cols),
		table.WithRows(rows),
		table.WithFocused(true),
		table.WithStyles(worktreeTableStyles()),
		table.WithWidth(tableWidth(cols)),
	)
	view := renderWorktreeTable(tb, wts)
	stripped := ansi.Strip(view)

	for _, want := range []string{"[dirty]", "[primary] [merged]"} {
		if !strings.Contains(stripped, want) {
			t.Fatalf("view missing full badge %q: %q", want, stripped)
		}
	}
	if strings.Contains(stripped, "[primar"+ellipsis) {
		t.Fatalf("view contains truncated primary badge: %q", stripped)
	}

	lines := strings.Split(view, "\n")
	if len(lines) < 3 {
		t.Fatalf("view has too few lines: %q", view)
	}
	if !strings.Contains(lines[2], "\x1b[") {
		t.Fatalf("dirty row is not styled: %q", lines[2])
	}
}
