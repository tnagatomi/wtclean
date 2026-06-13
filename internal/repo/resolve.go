package repo

import (
	"fmt"
	"os/exec"

	"github.com/tnagatomi/wtclean/internal/worktree"
)

// ResolvePrimaryDir reports the primary worktree path of the repository
// containing dir. git lists the main worktree first regardless of which
// worktree the command runs from, so resolution works identically from the
// primary checkout, a subdirectory of it, or inside a linked worktree. An
// error is returned when dir is not inside a git repository.
func ResolvePrimaryDir(dir string) (string, error) {
	out, err := exec.Command("git", "-C", dir, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return "", fmt.Errorf("not inside a git repository: %w", err)
	}
	wts := worktree.Parse(string(out))
	if len(wts) == 0 {
		return "", fmt.Errorf("no worktrees found for %q", dir)
	}
	return wts[0].Path, nil
}
