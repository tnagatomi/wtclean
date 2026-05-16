// Package worktree parses the porcelain output of `git worktree list`.
package worktree

import (
	"bufio"
	"strings"
	"time"
)

// Worktree describes a single entry from `git worktree list --porcelain`.
// Branch is stripped of its refs/heads/ prefix so callers can render it
// directly.
type Worktree struct {
	Path           string
	HEAD           string
	Branch         string
	Bare           bool
	Detached       bool
	Locked         bool
	LockReason     string
	Prunable       bool
	PrunableReason string

	// LastCommit is the commit time of HEAD. Parse leaves it zero; the
	// repo package populates it after parsing since porcelain output does
	// not carry commit metadata.
	LastCommit time.Time
}

// Parse reads the porcelain output of `git worktree list --porcelain` and
// returns the worktree entries in the order git emitted them. Unknown
// attribute lines are ignored so future git versions adding fields do not
// break parsing.
func Parse(s string) []Worktree {
	var (
		results []Worktree
		current *Worktree
	)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			if current != nil {
				results = append(results, *current)
				current = nil
			}
			continue
		}
		if current == nil {
			current = &Worktree{}
		}
		key, value, _ := strings.Cut(line, " ")
		switch key {
		case "worktree":
			current.Path = value
		case "HEAD":
			current.HEAD = value
		case "branch":
			current.Branch = strings.TrimPrefix(value, "refs/heads/")
		case "bare":
			current.Bare = true
		case "detached":
			current.Detached = true
		case "locked":
			current.Locked = true
			current.LockReason = value
		case "prunable":
			current.Prunable = true
			current.PrunableReason = value
		}
	}
	if current != nil {
		results = append(results, *current)
	}
	return results
}
