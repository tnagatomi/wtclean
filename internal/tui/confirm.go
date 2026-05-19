package tui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/tnagatomi/wtm/internal/deleter"
	"github.com/tnagatomi/wtm/internal/repo"
	"github.com/tnagatomi/wtm/internal/worktree"
)

// deleteCompleteMsg is dispatched from the goroutine that runs the
// deletion batch back into the bubbletea event loop. The TUI stays
// responsive while git is working because the slow exec calls live in
// the Cmd, not in Update.
type deleteCompleteMsg struct {
	failures []deleter.Failure
}

// warningBadges identifies badges that escalate a row in the confirmation
// list with a `⚠` prefix and add a per-kind line to the warnings block.
// Forcing the deletion is implied — there is no per-warning toggle.
var warningBadges = []worktree.Badge{
	worktree.BadgeDirty,
	worktree.BadgeUnpushed,
	worktree.BadgeLocked,
}

// warningMessages maps each warning badge to the wording shown in the
// confirmation summary. Kept here so the confirm view and any future help
// overlay describe the consequences in the same words.
var warningMessages = map[worktree.Badge]string{
	worktree.BadgeDirty:    "uncommitted changes will be lost",
	worktree.BadgeUnpushed: "commits not pushed will be lost",
	worktree.BadgeLocked:   "the lock will be released",
}

// enterConfirmDelete captures the currently selected worktrees as the
// deletion targets, derives the default state of the "Also delete
// branches" toggle, and switches to the confirmation screen. The toggle
// defaults ON only when every selected worktree is already merged; the
// spec treats any non-merged target as worth a deliberate opt-in.
func (m Model) enterConfirmDelete() Model {
	targets := make([]worktree.Worktree, 0, len(m.selected))
	for _, w := range m.worktreeSorted {
		if m.selected[w.Path] {
			targets = append(targets, w)
		}
	}
	m.deleteTargets = targets
	m.deleteBranchesToggle = allMerged(targets)
	m.screen = screenConfirmDelete
	return m
}

func (m Model) handleConfirmKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.screen = screenWorktrees
		return m, nil
	case "space":
		m.deleteBranchesToggle = !m.deleteBranchesToggle
		return m, nil
	case "y":
		return m, m.deleteCmd()
	}
	return m, nil
}

// deleteCmd returns a tea.Cmd so the long-running git invocations
// execute off the Update goroutine and the TUI stays responsive. The
// Cmd posts a deleteCompleteMsg with the collected failures back into
// Update.
func (m Model) deleteCmd() tea.Cmd {
	repoPath := m.repos[m.selectedRepoIdx].Path
	targets := m.deleteTargets
	alsoBranches := m.deleteBranchesToggle
	return func() tea.Msg {
		return deleteCompleteMsg{failures: deleter.Delete(repoPath, targets, alsoBranches)}
	}
}

// applyDeleteResult reloads the affected repo so the worktree list
// reflects the post-deletion state, then re-enters Screen 2 (which
// clears the selection and filter as a side-effect of enterWorktrees).
// Reload failures keep the stale repo data — the user can re-enter the
// repo from Screen 1 to retry.
func (m Model) applyDeleteResult(msg deleteCompleteMsg) Model {
	if r, err := repo.Load(m.repos[m.selectedRepoIdx].Path); err == nil {
		m.repos[m.selectedRepoIdx] = r
	}
	m = m.enterWorktrees(m.selectedRepoIdx)
	m.deleteFailures = msg.failures
	return m
}

func (m Model) confirmDeleteView() string {
	var b strings.Builder
	title := lipgloss.NewStyle().Bold(true).Render(fmt.Sprintf("Deleting %d worktrees:", len(m.deleteTargets)))
	fmt.Fprintf(&b, "%s\n\n", title)
	for _, w := range m.deleteTargets {
		prefix := "    "
		if w.HasAnyBadge(warningBadges) {
			prefix = "  ⚠ "
		}
		fmt.Fprintf(&b, "%s%s    %s\n", prefix, w.Path, renderBadges(w.Badges))
	}
	checkbox := "[ ]"
	if m.deleteBranchesToggle {
		checkbox = "[x]"
	}
	fmt.Fprintf(&b, "\nOptions:\n  %s Also delete branches\n", checkbox)
	if counts := badgeCounts(m.deleteTargets, warningBadges); len(counts) > 0 {
		b.WriteString("\n⚠ Warnings (deletion will be forced):\n")
		labelWidth := warningLabelWidth()
		for _, badge := range warningBadges {
			if n := counts[badge]; n > 0 {
				fmt.Fprintf(&b, "  - %-*s %s (%d)\n", labelWidth, badge.String()+":", warningMessages[badge], n)
			}
		}
	}
	help := faintStyle.Render("[y] Confirm    [n] Cancel    [space] toggle branches")
	fmt.Fprintf(&b, "\n%s\n", help)
	return b.String()
}

// warningLabelWidth returns the column width needed to align the longest
// "<badge>:" label across the warnings block. Derived rather than
// hard-coded so adding a new warning badge keeps the layout aligned.
func warningLabelWidth() int {
	w := 0
	for _, b := range warningBadges {
		if l := len(b.String()) + 1; l > w {
			w = l
		}
	}
	return w
}
