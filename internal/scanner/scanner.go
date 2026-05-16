// Package scanner walks configured root directories to discover git
// repositories. The walk prunes once a repository is found and does not
// follow symbolic links.
package scanner

import (
	"os"
	"path/filepath"
	"slices"
	"sync"
)

// Scan walks each root up to maxDepth and returns the discovered repository
// paths, deduplicated and sorted alphabetically. Inaccessible directories
// (permission denied, transient I/O errors) are skipped silently.
func Scan(roots []string, maxDepth int) ([]string, error) {
	perRoot := make([][]string, len(roots))
	errs := make([]error, len(roots))
	var wg sync.WaitGroup
	for i, root := range roots {
		wg.Go(func() {
			abs, err := filepath.Abs(root)
			if err != nil {
				errs[i] = err
				return
			}
			perRoot[i] = walkRoot(abs, maxDepth)
		})
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}
	seen := make(map[string]struct{})
	var repos []string
	for _, list := range perRoot {
		for _, p := range list {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				repos = append(repos, p)
			}
		}
	}
	slices.Sort(repos)
	return repos, nil
}

// walkRoot recursively walks root in parallel. Each directory level is
// processed by its own goroutine; siblings run concurrently so a single root
// containing many top-level repositories still benefits from parallelism.
func walkRoot(root string, maxDepth int) []string {
	var (
		mu    sync.Mutex
		repos []string
		wg    sync.WaitGroup
	)
	var walk func(path string, depth int)
	walk = func(path string, depth int) {
		if depth > maxDepth {
			return
		}
		isRepo, prune := classify(path)
		if isRepo {
			mu.Lock()
			repos = append(repos, path)
			mu.Unlock()
		}
		if prune {
			return
		}
		entries, err := os.ReadDir(path)
		if err != nil {
			return
		}
		for _, e := range entries {
			// DirEntry.IsDir reports false for symlinks (their type is
			// ModeSymlink, not ModeDir), so this check also handles the
			// "do not follow symbolic links" requirement.
			if !e.IsDir() {
				continue
			}
			sub := filepath.Join(path, e.Name())
			wg.Go(func() {
				walk(sub, depth+1)
			})
		}
	}
	wg.Go(func() { walk(root, 0) })
	wg.Wait()
	return repos
}

// classify inspects path and reports whether it is a repository and whether
// the walk should stop descending. Linked worktree directories (.git as a
// regular file) are not repositories themselves but still prune the walk so
// internal git artifacts are not traversed.
func classify(path string) (isRepo, prune bool) {
	gitPath := filepath.Join(path, ".git")
	if info, err := os.Lstat(gitPath); err == nil {
		if info.IsDir() {
			return true, true
		}
		if info.Mode().IsRegular() {
			return false, true
		}
	}
	if isBareRepo(path) {
		return true, true
	}
	return false, false
}

func isBareRepo(path string) bool {
	for _, marker := range []string{"HEAD", "objects", "refs"} {
		if _, err := os.Lstat(filepath.Join(path, marker)); err != nil {
			return false
		}
	}
	return true
}
