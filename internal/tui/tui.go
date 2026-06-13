// Package tui implements the bubbletea user interface for wtclean.
package tui

import (
	"fmt"
	"slices"
	"time"

	"charm.land/bubbles/v2/table"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/tnagatomi/wtclean/internal/deleter"
	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/scanner"
	"github.com/tnagatomi/wtclean/internal/worktree"
)

const (
	timeLayout = "2006-01-02 15:04:05"
	emptyTime  = "-"
	ellipsis   = "…"
	chromeRows = 4 // title + table header + help + trailing newline
)

// faintStyle is the dim-text style used for help lines and status
// summaries. Defined once so View() does not allocate a fresh style on
// every render.
var faintStyle = lipgloss.NewStyle().Faint(true)

type screenID int

const (
	screenRepos screenID = iota
	screenWorktrees
	screenConfirmDelete
)

type Model struct {
	repos []repo.Repo

	configPath   string
	scanRoots    []scanner.Root
	configSkip   []string
	totalScanned int

	screen          screenID
	selectedRepoIdx int

	// cwdMode is set when wtclean was launched with --cwd against a single
	// repository (the one containing the working directory). The repository
	// list does not exist in this mode, so esc on the worktree screen quits
	// rather than navigating back, and the help wording is adjusted to match.
	cwdMode bool

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
	deleteFailures       []deleter.Failure
	deleting             bool

	fetching   bool
	fetchError error

	scanning  bool
	scanError error

	helpVisible bool

	termWidth  int
	termHeight int
}

// ModelOptions carries config context the TUI needs to render
// empty-state messages and (later) error logs. Optional: NewModel can
// be called with a zero-value Options for tests that only care about
// the repo list.
type ModelOptions struct {
	ConfigPath   string
	ScanRoots    []scanner.Root
	Skip         []string
	TotalScanned int
}

func NewModel(repos []repo.Repo, opts ModelOptions) Model {
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
		repos:        repos,
		configPath:   opts.ConfigPath,
		scanRoots:    opts.ScanRoots,
		configSkip:   opts.Skip,
		totalScanned: opts.TotalScanned,
		screen:       screenRepos,
		repoTable:    t,
		repoMaxPath:  repoMaxPath,
	}
}

// NewScanning builds the model the production entry point starts with: no
// repos yet and scanning already set, so the first frame shows the scanning
// indicator and Init dispatches the discovery off the main goroutine. The
// repository list is then populated by the resulting scanCompleteMsg, sharing
// the exact code path the `r` refresh action uses.
func NewScanning(opts ModelOptions) Model {
	m := NewModel(nil, opts)
	m.scanning = true
	return m
}

// NewSingleRepo builds the model the --cwd entry point starts with: the one
// repository containing the working directory, opened directly on its worktree
// list. The repository-list screen is skipped entirely. Init dispatches no
// scan, so the worktree list is visible on the first frame; the delete-reload
// and fetch code paths still operate on this single repo unchanged.
func NewSingleRepo(r repo.Repo, opts ModelOptions) Model {
	m := NewModel([]repo.Repo{r}, opts)
	m.cwdMode = true
	return m.enterWorktrees(0)
}

// Init dispatches the initial scan when the model was constructed in the
// scanning state (the production path); a model seeded with repos for tests
// has nothing to load.
func (m Model) Init() tea.Cmd {
	if m.scanning {
		return m.scanCmd()
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.refreshLayout()
		return m, nil
	case tea.KeyPressMsg:
		return m.handleKey(msg)
	case deleteCompleteMsg:
		return m.applyDeleteResult(msg)
	case fetchCompleteMsg:
		return m.applyFetchResult(msg)
	case scanCompleteMsg:
		return m.applyScanResult(msg)
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
	if msg.String() == "?" {
		m.helpVisible = !m.helpVisible
		return m, nil
	}
	if m.helpVisible {
		if msg.String() == "esc" {
			m.helpVisible = false
		}
		return m, nil
	}
	switch m.screen {
	case screenRepos:
		switch msg.String() {
		case "enter":
			if len(m.repos) > 0 {
				m = m.enterWorktrees(m.repoTable.Cursor())
			}
			return m, nil
		case "r":
			if m.scanning {
				return m, nil
			}
			m.scanning = true
			m.scanError = nil
			return m, m.scanCmd()
		}
	case screenWorktrees:
		switch msg.String() {
		case "esc":
			if m.filterQuery != "" {
				return m.clearFilter(), nil
			}
			// In --cwd mode the worktree list is the top screen — there is
			// no repository list to return to — so esc quits.
			if m.cwdMode {
				return m, tea.Quit
			}
			m.screen = screenRepos
			return m, nil
		case "/":
			m.filterEditing = true
			return m, nil
		case "space":
			return m.toggleSelection(), nil
		case "s":
			return m.toggleSafeSelection(), nil
		case "d":
			if len(m.selected) > 0 {
				return m.enterConfirmDelete(), nil
			}
			return m, nil
		case "r":
			if m.fetching {
				return m, nil
			}
			m.fetching = true
			m.fetchError = nil
			return m, m.fetchCmd()
		}
	case screenConfirmDelete:
		return m.handleConfirmKey(msg)
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
	switch {
	case m.helpVisible:
		content = helpView(m.cwdMode)
	case m.screen == screenConfirmDelete:
		content = m.confirmDeleteView()
	case m.screen == screenWorktrees:
		content = m.worktreeView()
	default:
		content = m.repoView()
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}

func (m Model) repoView() string {
	title := lipgloss.NewStyle().Bold(true).Render("wtclean — repositories")
	// While the very first scan is still in flight there are no repos to
	// render yet; show only the indicator so the user doesn't see a flash
	// of the "no repositories" empty state before data arrives.
	if m.scanning && len(m.repos) == 0 {
		return fmt.Sprintf("%s\n%s\n", title, faintStyle.Render("⏳ Scanning..."))
	}
	// A scan failure that left us with no repos must show the error rather
	// than the "no repositories" empty state, which would misattribute the
	// failure to an empty filesystem.
	if m.scanError != nil && len(m.repos) == 0 {
		return fmt.Sprintf("%s\n%s\n\nPress r to retry, q to quit.\n", title, faintStyle.Render(fmt.Sprintf("⚠ scan failed: %v", m.scanError)))
	}
	if msg := m.repoEmptyMessage(); msg != "" {
		return msg + "\n\nPress q to quit.\n"
	}
	help := faintStyle.Render("[↑/k] up  [↓/j] down  [enter] open  [r] refresh  [?] help  [q] quit")
	status := ""
	switch {
	case m.scanning:
		status = "\n" + faintStyle.Render("⏳ Scanning...")
	case m.scanError != nil:
		status = "\n" + faintStyle.Render(fmt.Sprintf("⚠ scan failed: %v", m.scanError))
	}
	return fmt.Sprintf("%s\n%s\n%s%s\n", title, m.repoTable.View(), help, status)
}

// repoEmptyMessage returns the empty-state message for Screen 1, or ""
// when the screen has repos to render. Two cases survive past the
// pre-TUI config error path (config missing / zero roots already
// caught by config.Load):
//
//   - The scanner found NO git repositories under any configured root.
//   - The scanner found git repositories but every one of them only had
//     a primary worktree (no linked worktrees → all filtered out).
func (m Model) repoEmptyMessage() string {
	if len(m.repos) > 0 {
		return ""
	}

	if m.totalScanned == 0 {
		paths := make([]string, len(m.scanRoots))
		for i, r := range m.scanRoots {
			paths[i] = r.Path
		}
		return fmt.Sprintf("No repositories found under: %v\nConfig: %s", paths, m.configPath)
	}
	return "No worktrees found. (Repositories with only primary checkouts are hidden.)"
}

func (m Model) worktreeView() string {
	r := m.repos[m.selectedRepoIdx]
	titleText := fmt.Sprintf("wtclean — worktrees in %s", r.Path)
	if m.filterEditing || m.filterQuery != "" {
		cursor := ""
		if m.filterEditing {
			cursor = "_"
		}
		titleText += "    /" + m.filterQuery + cursor
	}
	title := lipgloss.NewStyle().Bold(true).Render(titleText)
	// esc clears the filter and then, in --cwd mode, quits (no repository
	// list to return to); otherwise it navigates back to the repository list.
	escHint := "[esc] back/clear"
	if m.cwdMode {
		escHint = "[esc] clear/quit"
	}
	help := faintStyle.Render("[↑/k] up  [↓/j] down  [space] select  [s] select safe  [/] filter  [d] delete  [r] fetch  [?] help  " + escHint + "  [q] quit")
	body := renderWorktreeTable(m.worktreeTable, m.worktreeVisible)
	if m.fetching {
		body += "\n" + faintStyle.Render("⏳ Fetching...")
	}
	if m.fetchError != nil {
		body += "\n" + faintStyle.Render(fmt.Sprintf("⚠ fetch failed: %v", m.fetchError))
	}
	if n := len(m.deleteFailures); n > 0 {
		body += "\n" + faintStyle.Render(fmt.Sprintf("⚠ %d operation(s) failed during last delete", n))
	}
	return fmt.Sprintf("%s\n%s\n%s\n", title, body, help)
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

// toggleSafeSelection toggles selection across the safe-to-remove worktrees
// that are currently visible. If every visible safe-to-remove worktree is
// already selected, they are all deselected; otherwise they are all selected.
// Worktrees outside that set — non-safe rows and rows hidden by the active
// filter — are never touched, so a manual selection survives a press of `s`.
// An empty visible safe set is a no-op.
func (m Model) toggleSafeSelection() Model {
	var safe []string
	for _, w := range m.worktreeVisible {
		if isSafeToRemove(w) {
			safe = append(safe, w.Path)
		}
	}
	if len(safe) == 0 {
		return m
	}
	allSelected := true
	for _, p := range safe {
		if !m.selected[p] {
			allSelected = false
			break
		}
	}
	for _, p := range safe {
		if allSelected {
			delete(m.selected, p)
		} else {
			m.selected[p] = true
		}
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
