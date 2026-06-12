// Package repo aggregates per-repository state used by the TUI: the
// worktrees enumerated by git plus the last fetch timestamp.
package repo

import (
	"iter"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/tnagatomi/wtclean/internal/scanner"
	"github.com/tnagatomi/wtclean/internal/worktree"
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
// below relies on that sentinel. totalScanned reports the number of git
// repositories the scanner found before the linked-count filter, so the
// caller can distinguish "no repos at all" from "all repos hidden because
// they only have a primary worktree".
func Discover(roots []string, maxDepth int, skip []string) (filtered []Repo, totalScanned int, err error) {
	paths, err := scanner.Scan(roots, maxDepth, skip)
	if err != nil {
		return nil, 0, err
	}
	repos := make([]Repo, len(paths))
	sem := make(chan struct{}, loadConcurrency)
	var wg sync.WaitGroup
	for i, p := range paths {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			r, err := Load(p)
			if err != nil {
				return
			}
			repos[i] = r
		})
	}
	wg.Wait()

	filtered = make([]Repo, 0, len(repos))
	for _, r := range repos {
		if r.Path != "" && r.LinkedCount() > 0 {
			filtered = append(filtered, r)
		}
	}
	slices.SortFunc(filtered, func(a, b Repo) int {
		return strings.Compare(a.Path, b.Path)
	})
	return filtered, len(paths), nil
}

// Load queries a single repository for its worktrees, badge state, and
// last-fetch timestamp. Discover uses it across the configured roots;
// the TUI uses it to refresh one repo after a delete or fetch.
func Load(path string) (Repo, error) {
	out, err := exec.Command("git", "-C", path, "worktree", "list", "--porcelain").Output()
	if err != nil {
		return Repo{}, err
	}
	wts := worktree.Parse(string(out))
	populateCommitTimes(path, wts)
	populateBadges(path, wts)
	return Repo{
		Path:      path,
		Worktrees: wts,
		LastFetch: fetchHeadMtime(path),
	}, nil
}

// populateCommitTimes fills LastCommit for every worktree HEAD using a
// single `git log --no-walk` invocation. One process per repo regardless of
// worktree count, and SHA-keyed output handles detached HEADs naturally.
// On any failure (corrupt HEAD, unknown SHA in the batch) the affected
// worktrees keep their zero LastCommit so callers render a placeholder.
func populateCommitTimes(repoPath string, wts []worktree.Worktree) {
	var shas []string
	for _, w := range wts {
		if w.HEAD != "" {
			shas = append(shas, w.HEAD)
		}
	}
	if len(shas) == 0 {
		return
	}
	args := append([]string{"-C", repoPath, "log", "--no-walk", "--pretty=%H %cI"}, shas...)
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return
	}
	times := make(map[string]time.Time, len(shas))
	for line := range gitLines(out) {
		sha, iso, ok := strings.Cut(line, " ")
		if !ok {
			continue
		}
		if t, err := time.Parse(time.RFC3339, iso); err == nil {
			times[sha] = t
		}
	}
	for i := range wts {
		if t, ok := times[wts[i].HEAD]; ok {
			wts[i].LastCommit = t
		}
	}
}

// gitLines iterates over the non-empty newline-trimmed lines of git command
// output. Centralizes the SplitSeq/TrimRight idiom shared by parsers in this
// package.
func gitLines(out []byte) iter.Seq[string] {
	return strings.SplitSeq(strings.TrimRight(string(out), "\n"), "\n")
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
