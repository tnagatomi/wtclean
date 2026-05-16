// Package repo aggregates per-repository state used by the TUI: the
// worktrees enumerated by git plus the last fetch timestamp.
package repo

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/tnagatomi/wtm/internal/scanner"
	"github.com/tnagatomi/wtm/internal/worktree"
)

type Repo struct {
	Path      string
	Worktrees []worktree.Worktree
	LastFetch time.Time
}

// LinkedCount returns the number of linked worktrees, i.e. all worktrees
// except the primary checkout. Git always lists the main worktree first,
// so any subsequent entries are linked.
func (r Repo) LinkedCount() int {
	if len(r.Worktrees) == 0 {
		return 0
	}
	return len(r.Worktrees) - 1
}

// loadConcurrency caps how many `git worktree list` invocations run in
// parallel. Each load spawns a git subprocess; the spec calls for at most 8
// concurrent per-repo git operations.
const loadConcurrency = 8

// Discover scans the configured roots, queries each repository's worktrees,
// reads the last fetch timestamp, and returns only repositories with at
// least one linked worktree, sorted alphabetically by path. Repositories
// whose git invocation fails are silently dropped — the Path-empty filter
// below relies on that sentinel.
func Discover(roots []string, maxDepth int) ([]Repo, error) {
	paths, err := scanner.Scan(roots, maxDepth)
	if err != nil {
		return nil, err
	}
	repos := make([]Repo, len(paths))
	sem := make(chan struct{}, loadConcurrency)
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			r, err := load(p)
			if err != nil {
				return
			}
			repos[i] = r
		})
	}
	wg.Wait()

	filtered := make([]Repo, 0, len(repos))
	for _, r := range repos {
		if r.Path != "" && r.LinkedCount() > 0 {
			filtered = append(filtered, r)
		}
	}
	slices.SortFunc(filtered, func(a, b Repo) int {
		return strings.Compare(a.Path, b.Path)
	})
	return filtered, nil
}

func load(path string) (Repo, error) {
	out, err := exec.Command("git", "-C", path, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return Repo{}, err
	}
	return Repo{
		Path:      path,
		Worktrees: worktree.Parse(string(out)),
		LastFetch: fetchHeadMtime(path),
	}, nil
}

// fetchHeadMtime reads FETCH_HEAD's mtime. For a non-bare checkout it lives
// under .git/, for a bare repo it lives directly under the repo path. A
// zero time is returned when FETCH_HEAD is absent (repo never fetched).
func fetchHeadMtime(path string) time.Time {
	for _, candidate := range []string{
		filepath.Join(path, ".git", "FETCH_HEAD"),
		filepath.Join(path, "FETCH_HEAD"),
	} {
		if info, err := os.Stat(candidate); err == nil {
			return info.ModTime()
		}
	}
	return time.Time{}
}
