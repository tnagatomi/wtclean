package tui

import (
	"strings"

	"github.com/tnagatomi/wtm/internal/worktree"
)

// renderBadges produces a plain, space-separated `[name]` list. Keep this
// value ANSI-free: bubbles/table truncates raw cell values before rendering,
// so embedded escape sequences are counted as content and can collapse the
// cell to an ellipsis under color-capable terminals.
func renderBadges(badges []worktree.Badge) string {
	if len(badges) == 0 {
		return ""
	}
	parts := make([]string, len(badges))
	for i, b := range badges {
		parts[i] = "[" + b.String() + "]"
	}
	return strings.Join(parts, " ")
}

// badgesVisibleWidth returns the longest rendered badges width across the
// given worktrees, honoring the header label so the column never collapses
// below "Badges" itself. Width is computed from the literal `[name]` shape
// so the layout stays independent from terminal color profile and ANSI
// escape handling.
func badgesVisibleWidth(wts []worktree.Worktree) int {
	w := len("Badges")
	for _, wt := range wts {
		if l := plainBadgesWidth(wt.Badges); l > w {
			w = l
		}
	}
	return w
}

// plainBadgesWidth returns the visible width of renderBadges as if no ANSI
// styling were applied. Layout depends on this for column sizing.
func plainBadgesWidth(badges []worktree.Badge) int {
	if len(badges) == 0 {
		return 0
	}
	total := 0
	for i, b := range badges {
		if i > 0 {
			total++ // single-space separator
		}
		total += 2 + len(b.String()) // "[" + name + "]"
	}
	return total
}
