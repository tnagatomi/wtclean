package tui

import (
	"errors"
	"os"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/repo"
	"github.com/tnagatomi/wtclean/internal/worktree"
	"github.com/tnagatomi/wtclean/internal/wtcleanlog"
)

// repoWith builds a repo at path carrying one linked worktree so it survives
// repo.Discover's linked-count filter and renders as a single row.
func repoWith(path string) repo.Repo {
	return repo.Repo{Path: path, Worktrees: []worktree.Worktree{{Path: path}, {Path: path + "/wt"}}}
}

// repoScreenModel returns a Model parked on Screen 1 (the repository list)
// with a window size set so the table viewport renders rows.
func repoScreenModel(t *testing.T, repos []repo.Repo) tea.Model {
	t.Helper()
	m := tea.Model(NewModel(repos, ModelOptions{}))
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 20})
	return m
}

func TestRKeyOnRepoListDispatchesScanAndMarksScanning(t *testing.T) {
	m := repoScreenModel(t, []repo.Repo{
		{Path: "/a", Worktrees: []worktree.Worktree{{Path: "/a"}, {Path: "/a/wt"}}},
	})
	updated, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd == nil {
		t.Fatal("r should dispatch a scan cmd")
	}
	if !updated.(Model).scanning {
		t.Error("r should mark Model.scanning = true so the view can show an indicator")
	}
}

func TestRWhileScanningIsNoOp(t *testing.T) {
	m := repoScreenModel(t, []repo.Repo{
		{Path: "/a", Worktrees: []worktree.Worktree{{Path: "/a"}, {Path: "/a/wt"}}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	if cmd != nil {
		t.Fatal("second r while scanning should be a no-op (got non-nil cmd)")
	}
}

func TestScanningIndicatorRendered(t *testing.T) {
	m := repoScreenModel(t, []repo.Repo{
		{Path: "/a", Worktrees: []worktree.Worktree{{Path: "/a"}, {Path: "/a/wt"}}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	view := m.(Model).View().Content
	if !strings.Contains(view, "Scanning") {
		t.Errorf("view should show a scanning indicator while m.scanning is true: %q", view)
	}
}

func TestScanCompleteClearsScanningAndReplacesRepos(t *testing.T) {
	m := repoScreenModel(t, []repo.Repo{
		{Path: "/a", Worktrees: []worktree.Worktree{{Path: "/a"}, {Path: "/a/wt"}}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	fresh := []repo.Repo{
		{Path: "/a", Worktrees: []worktree.Worktree{{Path: "/a"}, {Path: "/a/wt"}}},
		{Path: "/b", Worktrees: []worktree.Worktree{{Path: "/b"}, {Path: "/b/wt"}}},
	}
	m, _ = m.Update(scanCompleteMsg{repos: fresh, totalScanned: 2})
	got := m.(Model)
	if got.scanning {
		t.Error("scanning should be cleared after scanCompleteMsg")
	}
	if got.scanError != nil {
		t.Errorf("scanError should be nil on success: %v", got.scanError)
	}
	if len(got.repos) != 2 {
		t.Errorf("repos should be replaced with the freshly scanned list; got %d", len(got.repos))
	}
	if got.totalScanned != 2 {
		t.Errorf("totalScanned should be updated; got %d", got.totalScanned)
	}
}

func TestScanErrorSurfacedAndPreservesRepos(t *testing.T) {
	m := repoScreenModel(t, []repo.Repo{
		{Path: "/a", Worktrees: []worktree.Worktree{{Path: "/a"}, {Path: "/a/wt"}}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	m, _ = m.Update(scanCompleteMsg{err: errors.New("walk failed")})
	got := m.(Model)
	if got.scanning {
		t.Error("scanning should be cleared after a failed scan")
	}
	if got.scanError == nil {
		t.Error("scanError should be set after a failed scan")
	}
	if len(got.repos) != 1 {
		t.Errorf("existing repos should survive a failed scan; got %d", len(got.repos))
	}
	view := got.View().Content
	if !strings.Contains(view, "scan failed") || !strings.Contains(view, "walk failed") {
		t.Errorf("view should surface scan error: %q", view)
	}
}

func TestScanFailureLogged(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := repoScreenModel(t, []repo.Repo{
		{Path: "/a", Worktrees: []worktree.Worktree{{Path: "/a"}, {Path: "/a/wt"}}},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	_, cmd := m.Update(scanCompleteMsg{err: errors.New("walk failed")})
	if cmd == nil {
		t.Fatal("a failed scan should return a logging cmd")
	}
	cmd()
	path, err := wtcleanlog.Path()
	if err != nil {
		t.Fatalf("resolve log path: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read log: %v", err)
	}
	if !strings.Contains(string(data), "walk failed") {
		t.Errorf("scan failure should be recorded in the log; got %q", string(data))
	}
}

func TestRefreshPreservesCursorByPath(t *testing.T) {
	m := repoScreenModel(t, []repo.Repo{
		repoWith("/a"), repoWith("/b"), repoWith("/c"),
	})
	// Park the cursor on /b (index 1).
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	// A refresh inserts /aa before /b, shifting /b from index 1 to index 2.
	fresh := []repo.Repo{
		repoWith("/a"), repoWith("/aa"), repoWith("/b"), repoWith("/c"),
	}
	m, _ = m.Update(scanCompleteMsg{repos: fresh, totalScanned: 4})
	got := m.(Model)
	if at := got.repos[got.repoTable.Cursor()].Path; at != "/b" {
		t.Errorf("cursor should follow /b across the refresh; landed on %q", at)
	}
}

func TestRefreshClampsCursorWhenRepoGone(t *testing.T) {
	m := repoScreenModel(t, []repo.Repo{
		repoWith("/a"), repoWith("/b"), repoWith("/c"),
	})
	// Park the cursor on /c (index 2).
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	// /c disappears; only /a survives, so the cursor must clamp into range.
	m, _ = m.Update(scanCompleteMsg{repos: []repo.Repo{repoWith("/a")}, totalScanned: 1})
	got := m.(Model)
	if c := got.repoTable.Cursor(); c < 0 || c >= len(got.repos) {
		t.Errorf("cursor should clamp into the new range; got %d for %d repos", c, len(got.repos))
	}
}

func TestRefreshReconcilesSelectedRepoOnScreen2(t *testing.T) {
	m := repoScreenModel(t, []repo.Repo{
		repoWith("/a"), repoWith("/b"), repoWith("/c"),
	})
	// Open Screen 2 for /b, then a scan started on Screen 1 completes while
	// we're parked there — the list reorders, shifting /b to a new index.
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	fresh := []repo.Repo{
		repoWith("/a"), repoWith("/aa"), repoWith("/b"), repoWith("/c"),
	}
	m, _ = m.Update(scanCompleteMsg{repos: fresh, totalScanned: 4})
	got := m.(Model)
	if at := got.repos[got.selectedRepoIdx].Path; at != "/b" {
		t.Errorf("selectedRepoIdx should still resolve to the opened repo /b; got %q", at)
	}
	if !strings.Contains(got.View().Content, "/b") {
		t.Errorf("worktree view should still title the opened repo /b: %q", got.View().Content)
	}
}

func TestNewScanningStartsScanningAndInitDispatchesScan(t *testing.T) {
	m := NewScanning(ModelOptions{ConfigRoots: []string{"/x"}, MaxDepth: 2})
	if !m.scanning {
		t.Error("NewScanning should start in the scanning state")
	}
	if len(m.repos) != 0 {
		t.Errorf("NewScanning should start with no repos; got %d", len(m.repos))
	}
	if m.Init() == nil {
		t.Error("Init should dispatch the initial scan while scanning")
	}
}

func TestNewModelInitDoesNotScan(t *testing.T) {
	m := NewModel([]repo.Repo{repoWith("/a")}, ModelOptions{})
	if m.Init() != nil {
		t.Error("a model seeded with repos should not auto-scan on Init")
	}
}

func TestFirstScanFailureShownOnEmptyScreen(t *testing.T) {
	var m tea.Model = NewScanning(ModelOptions{ConfigRoots: []string{"/x"}})
	m, _ = m.Update(scanCompleteMsg{err: errors.New("walk failed")})
	view := m.(Model).View().Content
	if !strings.Contains(view, "scan failed") || !strings.Contains(view, "walk failed") {
		t.Errorf("a failed first scan should surface the error, not the empty state: %q", view)
	}
}

func TestScanCmdRunsDiscoverOverConfiguredRoots(t *testing.T) {
	// An empty root has no git repositories: discovery should succeed and
	// return an empty list, proving scanCmd wired repo.Discover to the
	// configured roots rather than silently doing nothing.
	m := NewScanning(ModelOptions{ConfigRoots: []string{t.TempDir()}, MaxDepth: 3})
	cmd := m.scanCmd()
	if cmd == nil {
		t.Fatal("scanCmd should return a non-nil tea.Cmd")
	}
	msg, ok := cmd().(scanCompleteMsg)
	if !ok {
		t.Fatalf("cmd should return a scanCompleteMsg, got %T", cmd())
	}
	if msg.err != nil {
		t.Errorf("scanning an empty root should not error: %v", msg.err)
	}
	if len(msg.repos) != 0 {
		t.Errorf("an empty root should discover no repos; got %d", len(msg.repos))
	}
}
