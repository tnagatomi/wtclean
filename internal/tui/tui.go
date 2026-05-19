// Package tui implements the bubbletea user interface for wtm.
package tui

import (
	"fmt"
	"slices"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/tnagatomi/wtm/internal/repo"
	"github.com/tnagatomi/wtm/internal/worktree"
)

const (
	timeLayout = "2006-01-02 15:04:05"
	emptyTime  = "-"
	ellipsis   = "…"
	chromeRows = 4 // title + table header + help + trailing newline
)

type screenID int

const (
	screenRepos screenID = iota
	screenWorktrees
	screenConfirmDelete
)

type Model struct {
	repos []repo.Repo

	screen          screenID
	selectedRepoIdx int

	repoTable   table.Model
	repoMaxPath int

	worktreeTable     table.Model
	worktreeSorted    []worktree.Worktree
	worktreeVisible   []worktree.Worktree
	worktreeMaxPath   int
	worktreeMaxBranch int
	worktreeMaxBadges int
	selected          map[string]bool

	filterEditing bool
	filterQuery   string

	deleteTargets        []worktree.Worktree
	deleteBranchesToggle bool

	termWidth  int
	termHeight int
}

func NewModel(repos []repo.Repo) Model {
	repoMaxPath := maxRepoPathWidth(repos)
	cols, rs := repoLayout(repos, repoMaxPath, 0)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rs),
		table.WithFocused(true),
		table.WithKeyMap(emacsTableKeyMap()),
		table.WithWidth(tableWidth(cols)),
	)
	return Model{
		repos:       repos,
		screen:      screenRepos,
		repoTable:   t,
		repoMaxPath: repoMaxPath,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.refreshLayout()
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	}
	return m.delegateToTable(msg)
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	// ctrl+c is always a quit; other shortcuts (including `q`) must not
	// fire while the filter input is consuming keypresses as literal text.
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.screen == screenWorktrees && m.filterEditing {
		return m.handleFilterEditKey(msg), nil
	}
	if msg.String() == "q" {
		return m, tea.Quit
	}
	switch m.screen {
	case screenRepos:
		if msg.String() == "enter" && len(m.repos) > 0 {
			m = m.enterWorktrees(m.repoTable.Cursor())
			return m, nil
		}
	case screenWorktrees:
		switch msg.String() {
		case "esc":
			if m.filterQuery != "" {
				return m.clearFilter(), nil
			}
			m.screen = screenRepos
			return m, nil
		case "/":
			m.filterEditing = true
			return m, nil
		case "space":
			return m.toggleSelection(), nil
		case "d":
			if len(m.selected) > 0 {
				return m.enterConfirmDelete(), nil
			}
			return m, nil
		}
	case screenConfirmDelete:
		return m.handleConfirmKey(msg), nil
	}
	return m.delegateToTable(msg)
}

func (m Model) delegateToTable(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch m.screen {
	case screenRepos:
		m.repoTable, cmd = m.repoTable.Update(msg)
	case screenWorktrees:
		m.worktreeTable, cmd = m.worktreeTable.Update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	var content string
	switch m.screen {
	case screenConfirmDelete:
		content = m.confirmDeleteView()
	case screenWorktrees:
		content = m.worktreeView()
	case screenRepos:
		content = m.repoView()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) repoView() string {
	if len(m.repos) == 0 {
		return "No repositories with linked worktrees found.\n\nPress q to quit.\n"
	}
	title := lipgloss.NewStyle().Bold(true).Render("wtm — repositories")
	help := lipgloss.NewStyle().Faint(true).Render("[↑/k] up  [↓/j] down  [enter] open  [q] quit")
	return fmt.Sprintf("%s\n%s\n%s\n", title, m.repoTable.View(), help)
}

func (m Model) worktreeView() string {
	r := m.repos[m.selectedRepoIdx]
	titleText := fmt.Sprintf("wtm — worktrees in %s", r.Path)
	if m.filterEditing || m.filterQuery != "" {
		cursor := ""
		if m.filterEditing {
			cursor = "_"
		}
		titleText += "    /" + m.filterQuery + cursor
	}
	title := lipgloss.NewStyle().Bold(true).Render(titleText)
	help := lipgloss.NewStyle().Faint(true).Render("[↑/k] up  [↓/j] down  [space] select  [/] filter  [esc] back/clear  [q] quit")
	return fmt.Sprintf("%s\n%s\n%s\n", title, renderWorktreeTable(m.worktreeTable, m.worktreeVisible), help)
}

// enterWorktrees switches to Screen 2 for the repo at idx. The worktree
// table is rebuilt from scratch on every entry so the column widths reflect
// the selected repo's data rather than carrying over from a previous repo.
func (m Model) enterWorktrees(idx int) Model {
	wts := sortWorktrees(m.repos[idx].Worktrees)
	maxPath, maxBranch := maxWorktreeWidths(wts)
	maxBadges := badgesVisibleWidth(wts)
	m.selected = map[string]bool{}
	m.filterEditing = false
	m.filterQuery = ""
	cols, rs := worktreeLayout(wts, m.selected, maxPath, maxBranch, maxBadges, m.termWidth)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rs),
		table.WithFocused(true),
		table.WithKeyMap(worktreeTableKeyMap()),
		table.WithStyles(worktreeTableStyles()),
		table.WithWidth(tableWidth(cols)),
	)
	// Defer SetHeight until WindowSizeMsg has populated termHeight;
	// otherwise the table would shrink to a single row before the first
	// resize event arrives (and tests would not see any rows render).
	if m.termHeight > 0 {
		t.SetHeight(max(1, m.termHeight-chromeRows))
	}
	m.worktreeTable = t
	m.worktreeSorted = wts
	m.worktreeVisible = wts
	m.worktreeMaxPath = maxPath
	m.worktreeMaxBranch = maxBranch
	m.worktreeMaxBadges = maxBadges
	m.selectedRepoIdx = idx
	m.screen = screenWorktrees
	return m
}

func (m *Model) refreshLayout() {
	switch m.screen {
	case screenRepos:
		cols, rs := repoLayout(m.repos, m.repoMaxPath, m.termWidth)
		m.repoTable.SetColumns(cols)
		m.repoTable.SetRows(rs)
		m.repoTable.SetWidth(tableWidth(cols))
		m.repoTable.SetHeight(max(1, m.termHeight-chromeRows))
	case screenWorktrees:
		cols, rs := worktreeLayout(m.worktreeVisible, m.selected, m.worktreeMaxPath, m.worktreeMaxBranch, m.worktreeMaxBadges, m.termWidth)
		m.worktreeTable.SetColumns(cols)
		m.worktreeTable.SetRows(rs)
		m.worktreeTable.SetWidth(tableWidth(cols))
		m.worktreeTable.SetHeight(max(1, m.termHeight-chromeRows))
	}
}

// sortWorktrees returns the worktrees ordered by HEAD commit time, oldest
// first, so the most stale entries (the typical cleanup targets) appear at
// the top. Worktrees with zero LastCommit (bare repos, lookup failures)
// sort before everything else.
func sortWorktrees(in []worktree.Worktree) []worktree.Worktree {
	out := slices.Clone(in)
	slices.SortFunc(out, func(a, b worktree.Worktree) int {
		return a.LastCommit.Compare(b.LastCommit)
	})
	return out
}

// repoLayout sizes the Path column to the longest actual repo path. The
// terminal width acts as an upper bound; under-cap, the column auto-fits.
func repoLayout(repos []repo.Repo, contentPathWidth, termWidth int) ([]table.Column, []table.Row) {
	const (
		countWidth = 9
		timeWidth  = len(timeLayout)
		padding    = 6
		minPath    = 20
	)
	pathWidth := max(contentPathWidth, minPath)
	if termWidth > 0 {
		pathWidth = max(minPath, min(pathWidth, termWidth-countWidth-timeWidth-padding))
	}
	cols := []table.Column{
		{Title: "Path", Width: pathWidth},
		{Title: "Worktrees", Width: countWidth},
		{Title: "Last fetch", Width: timeWidth},
	}
	rs := make([]table.Row, len(repos))
	for i, r := range repos {
		rs[i] = table.Row{
			truncateHead(r.Path, pathWidth),
			fmt.Sprintf("%d", r.LinkedCount()),
			formatTime(r.LastFetch),
		}
	}
	return cols, rs
}

// worktreeLayout sizes Path and Branch to their longest content, capped by
// the terminal width. When clamping is needed, Path absorbs the reduction
// first since it is typically the longest field. Badges and Last commit
// keep their natural widths since their content is fixed-shape.
func worktreeLayout(wts []worktree.Worktree, selected map[string]bool, contentPathWidth, contentBranchWidth, contentBadgesWidth, termWidth int) ([]table.Column, []table.Row) {
	const (
		timeWidth     = len(timeLayout)
		checkboxWidth = 3
		padding       = 12
		minPath       = 20
		minBranch     = 6
	)
	pathWidth := contentPathWidth
	branchWidth := contentBranchWidth
	// Badges are never clamped — their value is the whole point of the
	// column. Path absorbs any reduction first, then Branch, when the
	// terminal cannot fit Path's natural width.
	if termWidth > 0 {
		available := termWidth - timeWidth - checkboxWidth - contentBadgesWidth - padding
		if pathWidth+branchWidth > available {
			pathWidth = available - branchWidth
			if pathWidth < minPath {
				pathWidth = minPath
				branchWidth = max(minBranch, available-pathWidth)
			}
		}
	}
	pathWidth = max(pathWidth, minPath)
	branchWidth = max(branchWidth, minBranch)
	cols := []table.Column{
		{Title: "", Width: checkboxWidth},
		{Title: "Path", Width: pathWidth},
		{Title: "Branch", Width: branchWidth},
		{Title: "Last commit", Width: timeWidth},
		{Title: "Badges", Width: contentBadgesWidth},
	}
	rs := make([]table.Row, len(wts))
	for i, w := range wts {
		rs[i] = table.Row{
			checkboxCell(w, selected[w.Path]),
			truncateHead(w.Path, pathWidth),
			truncateHead(w.Branch, branchWidth),
			formatTime(w.LastCommit),
			renderBadges(w.Badges),
		}
	}
	return cols, rs
}

// toggleSelection flips the selection state on the focused worktree and
// rebuilds the visible rows so the checkbox column reflects the new state.
// Selection is keyed by worktree path so a row's selection survives the
// filter shrinking and re-expanding the visible set.
func (m Model) toggleSelection() Model {
	cursor := m.worktreeTable.Cursor()
	if cursor < 0 || cursor >= len(m.worktreeVisible) {
		return m
	}
	w := m.worktreeVisible[cursor]
	if !isSelectable(w) {
		return m
	}
	if m.selected[w.Path] {
		delete(m.selected, w.Path)
	} else {
		m.selected[w.Path] = true
	}
	_, rs := worktreeLayout(m.worktreeVisible, m.selected, m.worktreeMaxPath, m.worktreeMaxBranch, m.worktreeMaxBadges, m.termWidth)
	m.worktreeTable.SetRows(rs)
	return m
}

// tableWidth returns the natural viewport width needed to render every
// column of cols in full, including bubbles/table's default per-cell
// padding (one space on each side). bubbles/table v2 stores rows in an
// internal viewport that produces no output when its width is zero, so
// this width must be set explicitly at construction time and on every
// layout refresh — otherwise the table renders only its header.
func tableWidth(cols []table.Column) int {
	const cellPadding = 2
	w := 0
	for _, c := range cols {
		w += c.Width + cellPadding
	}
	return w
}

func maxRepoPathWidth(repos []repo.Repo) int {
	w := len("Path")
	for _, r := range repos {
		if l := len([]rune(r.Path)); l > w {
			w = l
		}
	}
	return w
}

func maxWorktreeWidths(wts []worktree.Worktree) (path, branch int) {
	path = len("Path")
	branch = len("Branch")
	for _, w := range wts {
		if l := len([]rune(w.Path)); l > path {
			path = l
		}
		if l := len([]rune(w.Branch)); l > branch {
			branch = l
		}
	}
	return
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return emptyTime
	}
	return t.Format(timeLayout)
}

// truncateHead returns s clipped to width runes, replacing the leading
// portion with an ellipsis when truncation is needed. The repository name
// at the tail of a path is the most informative segment, so we preserve it
// at the cost of the shared home/root prefix.
//
// Width is measured in runes, which assumes single-cell glyphs. Paths with
// East Asian wide characters can still overflow visually; if that becomes
// an issue, swap to go-runewidth (already in the dependency graph).
func truncateHead(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	if width == 1 {
		return ellipsis
	}
	return ellipsis + string(runes[len(runes)-width+1:])
}
