package tui

import (
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
