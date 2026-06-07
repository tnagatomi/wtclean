package tui

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/fetcher"
	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/wtcleanlog"
)

// fetchCompleteMsg is dispatched from the fetch goroutine back into the
// bubbletea event loop once both the network fetch and the post-fetch
// reload have finished. fetchErr captures a non-nil `git fetch` failure;
// reloaded carries the freshly-loaded repo when repo.Load succeeded
// (independently of whether fetch succeeded); loadErr captures a reload
// failure. Splitting the two errors lets the UI surface the most
// actionable one without conflating them.
type fetchCompleteMsg struct {
	fetchErr error
	reloaded *repo.Repo
	loadErr  error
}

// fetchCmd returns a tea.Cmd that runs fetch + reload off the Update
// goroutine and posts a fetchCompleteMsg back. Captures the repo path
// by value so the closure stays correct even if the Model index shifts
// while the fetch is in flight.
func (m Model) fetchCmd() tea.Cmd {
	repoPath := m.repos[m.selectedRepoIdx].Path
	return func() tea.Msg {
		fetchErr := fetcher.Fetch(repoPath)
		r, loadErr := repo.Load(repoPath)
		var reloaded *repo.Repo
		if loadErr == nil {
			reloaded = &r
		}
		return fetchCompleteMsg{fetchErr: fetchErr, reloaded: reloaded, loadErr: loadErr}
	}
}

// applyFetchResult merges a completed fetch back into the Model. When
// reload returned fresh data we replace the repo entry and re-enter
// Screen 2 so the new badges render; when reload failed (e.g. network
// blip while a fetch was racing a remove) we leave the existing repo
// and the user's Screen 2 context (selection, filter, cursor) intact
// so they don't silently lose state to a transient error.
//
// fetchErr takes precedence over loadErr in the surfaced summary since
// a fetch failure is usually the actionable one (network / auth issue);
// a load failure after a successful fetch is rarer and likely transient.
func (m Model) applyFetchResult(msg fetchCompleteMsg) (Model, tea.Cmd) {
	if msg.reloaded != nil {
		m.repos[m.selectedRepoIdx] = *msg.reloaded
		m = m.enterWorktrees(m.selectedRepoIdx)
	}
	m.fetching = false
	switch {
	case msg.fetchErr != nil:
		m.fetchError = msg.fetchErr
	case msg.loadErr != nil:
		m.fetchError = msg.loadErr
	default:
		m.fetchError = nil
	}
	return m, logFetchFailuresCmd(m.repos[m.selectedRepoIdx].Path, msg)
}

// logFetchFailuresCmd returns a tea.Cmd that writes one log line for
// each non-nil error in msg. fetch and reload are logged with distinct
// op labels so the user can tell which step failed when reading the
// log. Returns nil when both errors are nil.
func logFetchFailuresCmd(repoPath string, msg fetchCompleteMsg) tea.Cmd {
	if msg.fetchErr == nil && msg.loadErr == nil {
		return nil
	}
	return func() tea.Msg {
		if msg.fetchErr != nil {
			_ = wtcleanlog.Append(fmt.Sprintf("fetch:fetch %s: %v", repoPath, msg.fetchErr))
		}
		if msg.loadErr != nil {
			_ = wtcleanlog.Append(fmt.Sprintf("fetch:reload %s: %v", repoPath, msg.loadErr))
		}
		return nil
	}
}
