package tui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// worktreeEscDesc and worktreeEscDescCwd are the two phrasings of the
// worktree-list esc binding. In --cwd mode the worktree list is the top
// screen, so esc quits rather than returning to a (nonexistent) repository
// list; helpView swaps the wording to match the cwdMode esc behavior.
const (
	worktreeEscDesc    = "clear filter, or back to Screen 1"
	worktreeEscDescCwd = "clear filter, or quit"
)

// helpView renders the modal help overlay. The content is a single static
// reference grouped by screen, since the global `?` toggle shows the same
// overlay regardless of where the user pressed it. cwdMode adjusts the
// worktree esc wording to match --cwd mode, where esc quits.
func helpView(cwdMode bool) string {
	var b strings.Builder
	b.WriteString(lipgloss.NewStyle().Bold(true).Render("wtclean — keyboard reference"))
	b.WriteString("\n\n")
	for _, g := range helpGroups {
		b.WriteString(lipgloss.NewStyle().Bold(true).Render(g.title))
		b.WriteString("\n")
		for _, e := range g.entries {
			desc := e.desc
			if cwdMode && desc == worktreeEscDesc {
				desc = worktreeEscDescCwd
			}
			b.WriteString("  ")
			b.WriteString(e.keys)
			b.WriteString(strings.Repeat(" ", helpKeyColumn-len(e.keys)))
			b.WriteString(desc)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString(faintStyle.Render("[?] close help    [q] quit"))
	b.WriteString("\n")
	return b.String()
}

const helpKeyColumn = 28

type helpEntry struct {
	keys string
	desc string
}

type helpGroup struct {
	title   string
	entries []helpEntry
}

var helpGroups = []helpGroup{
	{
		title: "Global",
		entries: []helpEntry{
			{"?", "toggle this help"},
			{"q  /  ctrl+c", "quit"},
		},
	},
	{
		title: "Repository list (Screen 1)",
		entries: []helpEntry{
			{"↑ / k  /  ctrl+p", "previous repo"},
			{"↓ / j  /  ctrl+n", "next repo"},
			{"g / G", "jump to top / bottom"},
			{"enter", "open repo (go to Screen 2)"},
			{"r", "refresh the repository list (local rescan)"},
		},
	},
	{
		title: "Worktree list (Screen 2)",
		entries: []helpEntry{
			{"↑ / k  /  ctrl+p", "previous worktree"},
			{"↓ / j  /  ctrl+n", "next worktree"},
			{"ctrl+v / alt+v", "page down / up"},
			{"g / G", "jump to top / bottom"},
			{"space", "toggle selection on focused row"},
			{"s", "select all safe-to-remove worktrees"},
			{"/", "start incremental filter"},
			{"d", "open delete confirmation"},
			{"r", "fetch this repo and reload"},
			{"esc", worktreeEscDesc},
		},
	},
	{
		title: "Filter (while editing)",
		entries: []helpEntry{
			{"<printable>", "append to query"},
			{"backspace", "remove last character"},
			{"enter", "apply filter and exit edit"},
			{"esc", "clear filter and exit edit"},
		},
	},
	{
		title: "Delete confirmation (Screen 3)",
		entries: []helpEntry{
			{"y", "confirm deletion"},
			{"n  /  esc", "cancel back to Screen 2"},
			{"space", "toggle [Also delete branches]"},
		},
	},
	{
		title: "Badges — safe to remove",
		entries: []helpEntry{
			{"[merged]", "branch is merged into the default branch"},
			{"[upstream-gone]", "the branch's upstream was deleted (often a merged PR)"},
			{"[no-dir]", "the working directory is already gone"},
		},
	},
	{
		title: "Badges — removal loses local work",
		entries: []helpEntry{
			{"[uncommitted]", "has changes not yet committed"},
			{"[unpushed]", "has commits not yet pushed"},
			{"[locked]", "deliberately protected with a worktree lock"},
			{"[primary]", "the main checkout; never selectable"},
		},
	},
}
