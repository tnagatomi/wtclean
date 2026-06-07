package tui

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/wtcleanlog"
)

// scanCompleteMsg is dispatched from the scan goroutine back into the
// bubbletea event loop once repo.Discover has finished. On success repos
// carries the freshly discovered list and totalScanned the pre-filter count
// (mirroring repo.Discover's return); err captures a discovery failure, in
// which case repos is left nil so the handler can keep the existing list.
type scanCompleteMsg struct {
	repos        []repo.Repo
	totalScanned int
	err          error
}

// scanCmd returns a tea.Cmd that runs repo.Discover off the Update goroutine
// and posts a scanCompleteMsg back. It re-uses the roots and depth captured
// at startup — refresh re-derives the repository list from the filesystem but
// never re-reads config.
func (m Model) scanCmd() tea.Cmd {
	roots := m.configRoots
	depth := m.configDepth
	return func() tea.Msg {
		repos, totalScanned, err := repo.Discover(roots, depth)
		return scanCompleteMsg{repos: repos, totalScanned: totalScanned, err: err}
	}
}

// applyScanResult merges a completed scan back into the Model. On failure the
// existing repository list is kept untouched so a transient discovery error
// doesn't wipe the screen; the error is surfaced inline by repoView. On
// success the freshly discovered list replaces the old one and the repo table
// is rebuilt.
func (m Model) applyScanResult(msg scanCompleteMsg) (Model, tea.Cmd) {
	m.scanning = false
	if msg.err != nil {
		m.scanError = msg.err
		return m, logScanFailureCmd(msg.err)
	}
	m.scanError = nil

	// Capture, by path, the rows the user is anchored to before the list
	// changes, so we can follow them to their new positions: the Screen 1
	// cursor, and — if the user has drilled into Screen 2 while the scan was
	// in flight — the opened repo behind selectedRepoIdx. Both anchors are
	// "" on the very first scan (empty list), which resolveIndex maps to the
	// top.
	oldCursor := m.repoTable.Cursor()
	cursorAnchor := pathAt(m.repos, oldCursor)
	openedAnchor := pathAt(m.repos, m.selectedRepoIdx)

	m.repos = msg.repos
	m.totalScanned = msg.totalScanned
	m.repoMaxPath = maxRepoPathWidth(m.repos)
	m.refreshLayout()
	m.repoTable.SetCursor(resolveIndex(m.repos, cursorAnchor, oldCursor))
	m.selectedRepoIdx = resolveIndex(m.repos, openedAnchor, m.selectedRepoIdx)
	return m, nil
}

// pathAt returns the repo path at idx, or "" when idx is out of range.
func pathAt(repos []repo.Repo, idx int) string {
	if idx < 0 || idx >= len(repos) {
		return ""
	}
	return repos[idx].Path
}

// resolveIndex re-locates a row in the rebuilt repository list after a
// refresh. anchor is the path the position was bound to before the scan ("" on
// the first scan, when nothing was bound); oldIdx is its former index. When
// the anchored repository still exists its new index is returned; otherwise
// the old index is clamped into the new list's bounds (0 when the list is
// empty). Used for both the Screen 1 cursor and the Screen 2 selectedRepoIdx.
func resolveIndex(repos []repo.Repo, anchor string, oldIdx int) int {
	if i := slices.IndexFunc(repos, func(r repo.Repo) bool { return r.Path == anchor }); i >= 0 {
		return i
	}
	if len(repos) == 0 {
		return 0
	}
	return min(oldIdx, len(repos)-1)
}

// logScanFailureCmd returns a tea.Cmd that records a single scan-failure line
// to the wtclean log. The on-screen summary in repoView is intentionally
// terse; the full error text is preserved here for later debugging. Mirrors
// logFetchFailuresCmd.
func logScanFailureCmd(err error) tea.Cmd {
	return func() tea.Msg {
		_ = wtcleanlog.Append(fmt.Sprintf("scan: %v", err))
		return nil
	}
}
