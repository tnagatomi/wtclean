// Package tui implements the bubbletea user interface for wtm.
package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/tnagatomi/wtm/internal/repo"
)

const (
	timeLayout = "2006-01-02 15:04:05"
	emptyTime  = "-"
	ellipsis   = "…"
)

type Model struct {
	repos            []repo.Repo
	table            table.Model
	contentPathWidth int
}

func NewModel(repos []repo.Repo) Model {
	cpw := maxPathWidth(repos)
	cols, rs := layout(repos, cpw, 0)
	t := table.New(
		table.WithColumns(cols),
		table.WithRows(rs),
		table.WithFocused(true),
	)
	return Model{repos: repos, table: t, contentPathWidth: cpw}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cols, rs := layout(m.repos, m.contentPathWidth, msg.Width)
		m.table.SetColumns(cols)
		m.table.SetRows(rs)
		// Header line, table header, help line, and a trailing newline
		// each consume one row, so leave four rows for chrome.
		m.table.SetHeight(max(1, msg.Height-4))
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if len(m.repos) == 0 {
		return "No repositories with linked worktrees found.\n\nPress q to quit.\n"
	}
	title := lipgloss.NewStyle().Bold(true).Render("wtm — repositories")
	help := lipgloss.NewStyle().Faint(true).Render("[↑/k] up  [↓/j] down  [q] quit")
	return fmt.Sprintf("%s\n%s\n%s\n", title, m.table.View(), help)
}

// layout computes the table columns and rows for a given terminal width.
// Both are recomputed together because the Path column's effective width
// drives how individual path strings are truncated.
func layout(repos []repo.Repo, contentPathWidth, termWidth int) ([]table.Column, []table.Row) {
	cols := columns(termWidth, contentPathWidth)
	return cols, rows(repos, cols[0].Width)
}

// columns sizes the Path column to fit the longest actual path rather than
// expanding to fill the terminal — a full-terminal-wide table forces the
// user's eyes to track across whitespace. The terminal width only acts as
// an upper bound when the content would otherwise overflow.
func columns(termWidth, contentPathWidth int) []table.Column {
	const (
		countWidth = 9
		timeWidth  = len(timeLayout)
		padding    = 6
		minPath    = 20
	)
	pathWidth := contentPathWidth
	if pathWidth < minPath {
		pathWidth = minPath
	}
	if termWidth > 0 {
		available := termWidth - countWidth - timeWidth - padding
		if pathWidth > available {
			pathWidth = available
		}
		if pathWidth < minPath {
			pathWidth = minPath
		}
	}
	return []table.Column{
		{Title: "Path", Width: pathWidth},
		{Title: "Worktrees", Width: countWidth},
		{Title: "Last fetch", Width: timeWidth},
	}
}

func rows(repos []repo.Repo, pathWidth int) []table.Row {
	out := make([]table.Row, len(repos))
	for i, r := range repos {
		out[i] = table.Row{
			truncateHead(r.Path, pathWidth),
			fmt.Sprintf("%d", r.LinkedCount()),
			formatTime(r.LastFetch),
		}
	}
	return out
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return emptyTime
	}
	return t.Format(timeLayout)
}

func maxPathWidth(repos []repo.Repo) int {
	w := len("Path")
	for _, r := range repos {
		if l := len([]rune(r.Path)); l > w {
			w = l
		}
	}
	return w
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
