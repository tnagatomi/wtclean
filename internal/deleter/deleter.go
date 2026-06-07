// Package deleter executes a batch worktree-removal plan against a single
// repository. Per-target failures are accumulated rather than aborting,
// matching the spec's continue-on-error rule.
package deleter

import (
	"fmt"
	"os/exec"
	"slices"

	"github.com/tnagatomi/wtclean/internal/worktree"
)

// Op names the git operation that failed; carried on Failure so callers
// can group / log failures by kind without parsing the error message.
type Op string

const (
	OpUnlock Op = "unlock"
	OpRemove Op = "remove"
	OpPrune  Op = "prune"
	OpBranch Op = "branch"
)

// Failure records a single failed git invocation. Path is the affected
// worktree (or the repo path for prune); Op is the operation kind that
// failed; Err carries the command error including any captured stderr.
type Failure struct {
	Path string
	Op   Op
	Err  error
}

func (f Failure) Error() string {
	return fmt.Sprintf("%s %s: %v", f.Op, f.Path, f.Err)
}

// forceRemoveBadges trigger `git worktree remove --force`. The spec calls
// these out as the cases where deletion will be forced. Kept in sync
// with the TUI's warningBadges (tui/confirm.go) by convention — the two
// happen to coincide today but are conceptually distinct (one drives
// --force, the other drives the warnings block on the confirm screen).
var forceRemoveBadges = []worktree.Badge{
	worktree.BadgeUncommitted,
	worktree.BadgeUnpushed,
	worktree.BadgeLocked,
}

// Delete runs the configured deletion plan against targets in repoPath.
// Each target is treated independently; a failure on one does not stop
// the rest. A single `git worktree prune` is appended once at the end
// when any target carried [no-dir], since prune is repo-wide rather
// than per-path.
func Delete(repoPath string, targets []worktree.Worktree, alsoBranches bool) []Failure {
	var failures []Failure
	anyNoDir := false
	for _, w := range targets {
		if slices.Contains(w.Badges, worktree.BadgeNoDir) {
			anyNoDir = true
			continue
		}
		if slices.Contains(w.Badges, worktree.BadgeLocked) {
			if err := run(repoPath, "worktree", "unlock", w.Path); err != nil {
				failures = append(failures, Failure{Path: w.Path, Op: OpUnlock, Err: err})
				// Continue to remove anyway — the force flag may still succeed.
			}
		}
		args := []string{"worktree", "remove", w.Path}
		if w.HasAnyBadge(forceRemoveBadges) {
			args = []string{"worktree", "remove", "--force", w.Path}
		}
		if err := run(repoPath, args...); err != nil {
			failures = append(failures, Failure{Path: w.Path, Op: OpRemove, Err: err})
			continue
		}
		if alsoBranches && w.Branch != "" {
			if err := deleteBranch(repoPath, w); err != nil {
				failures = append(failures, Failure{Path: w.Path, Op: OpBranch, Err: err})
			}
		}
	}
	if anyNoDir {
		if err := run(repoPath, "worktree", "prune"); err != nil {
			failures = append(failures, Failure{Path: repoPath, Op: OpPrune, Err: err})
		}
	}
	return failures
}

func deleteBranch(repoPath string, w worktree.Worktree) error {
	flag := "-D"
	if slices.Contains(w.Badges, worktree.BadgeMerged) {
		flag = "-d"
	}
	return run(repoPath, "branch", flag, w.Branch)
}

func run(repoPath string, args ...string) error {
	full := append([]string{"-C", repoPath}, args...)
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}
