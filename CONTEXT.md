# wtclean

A TUI tool for listing and deleting stale git worktrees across many repositories.

## Language

### Worktree states (badges)

A worktree carries zero or more **badges** describing its state. Badges fall
into two roles for deletion: some signal a worktree is safe to remove, others
warn that removal would lose local-only work.

**Primary**:
The main checkout of a repository. Never selectable and never deletable.
_Avoid_: main worktree, root checkout

**Merged**:
The branch's commits are reachable from the repository's default branch in the
local repo — a local check (`git branch --merged`). Misses squash- or
rebase-merged branches, whose commits land on the default branch under new
SHAs; _upstream-gone_ covers those.

**Upstream-gone**:
The branch's upstream tracking branch has been deleted — a check against the
remote, not the local default branch. Usually means the branch was merged via a
squash or rebase PR and its head was auto-deleted, so it catches merges that
_merged_ cannot see. Does not apply to a branch that never had an upstream.
_Avoid_: gone, orphaned

**Uncommitted**:
The worktree has changes that are not yet committed. Removing it loses that
work irrecoverably.
_Avoid_: dirty

**Unpushed**:
The branch has local commits that have not been pushed. Removing it loses
those commits irrecoverably.

**Locked**:
The worktree has been deliberately protected with a worktree lock.

**No-dir**:
The worktree's working directory has been deleted by hand; only an
administrative record of it remains.
_Avoid_: missing, prunable

### Data freshness

Two distinct user actions bring the displayed state up to date. They differ in
scope (all repositories vs. one) and in whether they touch the network.

**Refresh**:
Re-derive the whole repository list from the filesystem: rescan the configured
roots and re-read every repository's worktrees and badges. Picks up
repositories and worktrees created or deleted outside the app since startup.
Local only — it never contacts a remote. The repository-list screen's action.
_Avoid_: reload, rescan

**Fetch**:
Contact the remote for a single repository, then re-read that one repository's
state. Network-bound; updates remote-derived badges such as _upstream-gone_.
The worktree screen's action. Distinct from _refresh_, which is local and
spans every repository.

### Selection

**Safe-to-remove**:
A selectable (non-_primary_) worktree that carries positive evidence it is
disposable — at least one of _merged_, _upstream-gone_, or _no-dir_ — and none
of the warning states _uncommitted_, _unpushed_, or _locked_. A clean, pushed,
but unmerged worktree is not safe-to-remove: absence of warnings is not evidence
of disposability.
