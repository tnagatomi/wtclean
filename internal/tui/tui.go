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
)

type Model struct {
	repos []repo.Repo
	table table.Model
}

func NewModel(repos []repo.Repo) Model {
	t := table.New(
		table.WithColumns(columns(0)),
		table.WithRows(rows(repos)),
		table.WithFocused(true),
	)
	return Model{repos: repos, table: t}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetColumns(columns(msg.Width))
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

// columns sizes the path column to fill the remaining terminal width. Before
// the first WindowSizeMsg, width is 0 and we fall back to a generous default
// so the initial render is readable. For narrow terminals we clamp to a
// minimum rather than overflowing.
func columns(width int) []table.Column {
	const (
		countWidth = 12
		timeWidth  = len(timeLayout)
		padding    = 6
	)
	var pathWidth int
	switch {
	case width == 0:
		pathWidth = 80
	default:
		pathWidth = width - countWidth - timeWidth - padding
		if pathWidth < 20 {
			pathWidth = 20
		}
	}
	return []table.Column{
		{Title: "Path", Width: pathWidth},
		{Title: "Worktrees", Width: countWidth},
		{Title: "Last fetch", Width: timeWidth},
	}
}

func rows(repos []repo.Repo) []table.Row {
	out := make([]table.Row, len(repos))
	for i, r := range repos {
		out[i] = table.Row{
			r.Path,
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
