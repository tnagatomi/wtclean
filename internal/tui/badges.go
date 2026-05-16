package tui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/table"
	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/compat"

	"github.com/tnagatomi/wtm/internal/worktree"
)

var worktreeRowStyles = map[worktree.Badge]lipgloss.Style{
	worktree.BadgePrimary:  lipgloss.NewStyle().Foreground(adaptive("240", "245")),
	worktree.BadgeMerged:   lipgloss.NewStyle().Foreground(adaptive("28", "82")),
	worktree.BadgeGone:     lipgloss.NewStyle().Foreground(adaptive("130", "220")),
	worktree.BadgeDirty:    lipgloss.NewStyle().Foreground(adaptive("160", "203")),
	worktree.BadgeUnpushed: lipgloss.NewStyle().Foreground(adaptive("160", "203")),
	worktree.BadgeLocked:   lipgloss.NewStyle().Foreground(adaptive("93", "141")),
	worktree.BadgeMissing:  lipgloss.NewStyle().Foreground(adaptive("240", "245")),
}

// adaptive returns a lipgloss color that resolves to light when the
// terminal background is light and dark when it is dark. lipgloss v2
// moved AdaptiveColor into the compat package and switched to typed
// color.Color values, so palette declarations need the helper to stay
// readable.
func adaptive(light, dark string) compat.AdaptiveColor {
	return compat.AdaptiveColor{Light: lipgloss.Color(light), Dark: lipgloss.Color(dark)}
}

// worktreeRowBadgePriority orders badges by how much each should dominate
// a row's foreground color: action-required states first (dirty/unpushed/
// locked/gone/missing), then informational ones (primary/merged). The
// first match in this slice wins.
var worktreeRowBadgePriority = []worktree.Badge{
	worktree.BadgeDirty,
	worktree.BadgeUnpushed,
	worktree.BadgeLocked,
	worktree.BadgeGone,
	worktree.BadgeMissing,
	worktree.BadgePrimary,
	worktree.BadgeMerged,
}

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

func worktreeTableStyles() table.Styles {
	styles := table.DefaultStyles()
	styles.Selected = lipgloss.NewStyle().Bold(true)
	return styles
}

// renderWorktreeTable post-processes t.View() to tint each rendered data
// line by the highest-priority badge of its worktree. bubbles/table v1.0.0
// does not expose its viewport YOffset, so the first visible data row is
// approximated from the cursor and table height; this is exact when the
// worktree list fits in the terminal and best-effort when scrolled. The
// cursor row's bold styling is handled by the table's Selected style and
// is not reapplied here.
func renderWorktreeTable(t table.Model, wts []worktree.Worktree) string {
	view := t.View()
	lines := strings.Split(view, "\n")
	if len(lines) <= 1 {
		return view
	}
	height := t.Height()
	cursor := t.Cursor()
	yoffset := 0
	if len(wts) > height && cursor >= height {
		yoffset = min(cursor-height+1, len(wts)-height)
	}
	for lineIdx := 1; lineIdx < len(lines); lineIdx++ {
		rowIdx := yoffset + lineIdx - 1
		if rowIdx >= len(wts) {
			break
		}
		lines[lineIdx] = rowStyleForBadges(wts[rowIdx].Badges).Render(lines[lineIdx])
	}
	return strings.Join(lines, "\n")
}

func rowStyleForBadges(badges []worktree.Badge) lipgloss.Style {
	for _, badge := range worktreeRowBadgePriority {
		if slices.Contains(badges, badge) {
			return worktreeRowStyles[badge]
		}
	}
	return lipgloss.NewStyle()
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
