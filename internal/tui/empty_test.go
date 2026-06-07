package tui

import (
	"strings"
	"testing"

	"github.com/tnagatomi/wtclean/internal/repo"
)

func TestRepoEmptyMessage_NoScannedRepos(t *testing.T) {
	// Case 3: scanner found NO git repos under any configured root.
	m := NewModel(nil, ModelOptions{
		ConfigPath:   "/home/u/.config/wtclean/config.yml",
		ConfigRoots:  []string{"/home/u/src", "/home/u/work"},
		TotalScanned: 0,
	})
	view := m.View().Content
	if !strings.Contains(view, "No repositories found under") {
		t.Errorf("view should show 'No repositories found under' for the zero-scanned case: %q", view)
	}
	if !strings.Contains(view, "/home/u/src") {
		t.Errorf("view should list the configured roots: %q", view)
	}
	if !strings.Contains(view, "/home/u/.config/wtclean/config.yml") {
		t.Errorf("view should reference the config path: %q", view)
	}
}

func TestRepoEmptyMessage_AllRepoFilteredOut(t *testing.T) {
	// Case 4: scanner found git repos but every one of them only had a
	// primary worktree, so the filter dropped them all.
	m := NewModel(nil, ModelOptions{
		ConfigPath:   "/home/u/.config/wtclean/config.yml",
		ConfigRoots:  []string{"/home/u/src"},
		TotalScanned: 5,
	})
	view := m.View().Content
	if !strings.Contains(view, "No worktrees found") {
		t.Errorf("view should show 'No worktrees found' when all repos were filtered: %q", view)
	}
	if !strings.Contains(view, "only primary checkouts") {
		t.Errorf("view should explain why the list is empty: %q", view)
	}
}

func TestRepoEmptyMessage_HasRepos(t *testing.T) {
	// When there are repos to render, repoEmptyMessage should return ""
	// so the table renders instead.
	m := NewModel([]repo.Repo{{Path: "/repo"}}, ModelOptions{TotalScanned: 1})
	if got := m.repoEmptyMessage(); got != "" {
		t.Errorf("repoEmptyMessage should be empty when repos render: %q", got)
	}
}
