package tui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/tnagatomi/wtclean/internal/worktree"
)

// clipboardStub captures the last string handed to the setClipboard seam so
// tests can assert what would be copied without depending on bubbletea's
// unexported OSC52 message type.
type clipboardStub struct {
	got    string
	called bool
}

// stubClipboard swaps the package-level setClipboard seam for a capturing
// stub and restores the original when the test ends.
func stubClipboard(t *testing.T) *clipboardStub {
	t.Helper()
	orig := setClipboard
	s := &clipboardStub{}
	setClipboard = func(text string) tea.Cmd {
		s.got = text
		s.called = true
		return func() tea.Msg { return nil }
	}
	t.Cleanup(func() { setClipboard = orig })
	return s
}

func TestCopyBranchKeyCopiesFocusedBranch(t *testing.T) {
	clip := stubClipboard(t)
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/a", Branch: "feat-a"},
	})
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd == nil {
		t.Fatal("y on a branch row returned a nil cmd; expected a clipboard cmd")
	}
	if !clip.called {
		t.Fatal("y did not invoke the setClipboard seam")
	}
	if clip.got != "feat-a" {
		t.Fatalf("clipboard got %q; want %q", clip.got, "feat-a")
	}
}

func TestCopyBranchKeyIsNoOpOnBranchlessRow(t *testing.T) {
	clip := stubClipboard(t)
	// A detached-HEAD / bare worktree carries no branch name.
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/detached", Branch: ""},
	})
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if cmd != nil {
		t.Fatal("y on a branchless row returned a non-nil cmd; expected no-op")
	}
	if clip.called {
		t.Fatalf("y on a branchless row wrote %q to the clipboard; expected no write", clip.got)
	}
}

func TestCopyBranchSetsSuccessNotice(t *testing.T) {
	stubClipboard(t)
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/a", Branch: "feat-a"},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if got, want := m.(Model).copyNotice, "✓ Copied branch: feat-a"; got != want {
		t.Fatalf("copyNotice = %q; want %q", got, want)
	}
}

func TestCopyBranchSetsNoBranchNoticeOnBranchlessRow(t *testing.T) {
	stubClipboard(t)
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/detached", Branch: ""},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if got, want := m.(Model).copyNotice, "– No branch on this row"; got != want {
		t.Fatalf("copyNotice = %q; want %q", got, want)
	}
}

func TestCopyNoticeClearedByNextKey(t *testing.T) {
	stubClipboard(t)
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/a", Branch: "feat-a"},
		{Path: "/repo/wt/b", Branch: "feat-b"},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if m.(Model).copyNotice == "" {
		t.Fatal("precondition: y should have set a notice")
	}
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	if got := m.(Model).copyNotice; got != "" {
		t.Fatalf("copyNotice = %q after a non-y key; want it cleared", got)
	}
}

func TestCopyNoticeReplacedByRepeatedCopy(t *testing.T) {
	stubClipboard(t)
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/a", Branch: "feat-a"},
		{Path: "/repo/wt/b", Branch: "feat-b"},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}) // copies feat-a
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyDown})    // move to feat-b
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"}) // copies feat-b
	if got, want := m.(Model).copyNotice, "✓ Copied branch: feat-b"; got != want {
		t.Fatalf("copyNotice = %q; want %q", got, want)
	}
}

func TestCopyNoticeRenderedInView(t *testing.T) {
	stubClipboard(t)
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/a", Branch: "feat-a"},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: 'y', Text: "y"})
	if view := m.(Model).View().Content; !strings.Contains(view, "✓ Copied branch: feat-a") {
		t.Fatalf("view missing copy notice: %q", view)
	}
}

func TestCopyNoticeAndFetchErrorRenderTogether(t *testing.T) {
	stubClipboard(t)
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/a", Branch: "feat-a"},
	}).(Model)
	// The copy notice and fetch status are independent lines; setting both
	// must render both, not have one clobber the other.
	m.copyNotice = "✓ Copied branch: feat-a"
	m.fetchError = errors.New("boom")
	view := m.View().Content
	if !strings.Contains(view, "✓ Copied branch: feat-a") {
		t.Errorf("view missing copy notice: %q", view)
	}
	if !strings.Contains(view, "boom") {
		t.Errorf("view missing fetch error: %q", view)
	}
}

func TestWorktreeHelpLineListsCopyKey(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/a", Branch: "feat-a"},
	})
	if view := m.(Model).View().Content; !strings.Contains(view, "[y] copy branch name") {
		t.Fatalf("worktree help line missing copy-branch key: %q", view)
	}
}

func TestHelpOverlayDocumentsCopyKey(t *testing.T) {
	m := worktreeScreenModel(t, []worktree.Worktree{
		{Path: "/repo/wt/a", Branch: "feat-a"},
	})
	m, _ = m.Update(tea.KeyPressMsg{Code: '?', Text: "?"})
	if view := m.(Model).View().Content; !strings.Contains(view, "copy focused branch name to clipboard") {
		t.Fatalf("help overlay missing copy-branch entry: %q", view)
	}
}
