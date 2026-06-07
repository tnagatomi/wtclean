package repo

import (
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/tnagatomi/wtclean/internal/worktree"
)

// populateBadges fills Worktree.Badges for every entry in wts. Badge
// detection is intentionally local-only: no `git fetch` is invoked, so
// `merged` and `upstream-gone` reflect the most recent fetch state — the spec
// surfaces that staleness via the Screen 1 last-fetch column.
func populateBadges(repoPath string, wts []worktree.Worktree) {
	if len(wts) == 0 {
		return
	}

	noDir := make([]bool, len(wts))
	for i := range wts {
		// Git lists the main worktree first.
		if i == 0 {
			wts[i].Badges = append(wts[i].Badges, worktree.BadgePrimary)
		}
		if wts[i].Locked {
			wts[i].Badges = append(wts[i].Badges, worktree.BadgeLocked)
		}
		noDir[i] = wts[i].Prunable || !dirExists(wts[i].Path)
		if noDir[i] {
			wts[i].Badges = append(wts[i].Badges, worktree.BadgeNoDir)
		}
	}

	merged := mergedBranches(repoPath)
	track := branchTracking(repoPath)
	for i := range wts {
		if wts[i].Branch == "" {
			continue
		}
		if merged[wts[i].Branch] {
			wts[i].Badges = append(wts[i].Badges, worktree.BadgeMerged)
		}
		if t, ok := track[wts[i].Branch]; ok {
			if t.gone {
				wts[i].Badges = append(wts[i].Badges, worktree.BadgeUpstreamGone)
			}
			if t.ahead > 0 {
				wts[i].Badges = append(wts[i].Badges, worktree.BadgeUnpushed)
			}
		}
	}

	for i := range wts {
		if noDir[i] || wts[i].Bare {
			continue
		}
		if isDirty(wts[i].Path) {
			wts[i].Badges = append(wts[i].Badges, worktree.BadgeUncommitted)
		}
	}
}

func dirExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && info.IsDir()
}

// mergedBranches returns the set of local branches fully merged into the
// repository's default branch. An empty map is returned when the default
// branch cannot be resolved (no remote HEAD, no main/master, etc.).
func mergedBranches(repoPath string) map[string]bool {
	def := defaultBranch(repoPath)
	if def == "" {
		return nil
	}
	out, err := exec.Command("git", "-C", repoPath, "branch", "--merged", def, "--format=%(refname:short)").Output()
	if err != nil {
		return nil
	}
	merged := make(map[string]bool)
	for line := range gitLines(out) {
		if line != "" {
			merged[line] = true
		}
	}
	return merged
}

// defaultBranch resolves the repository's default branch name by checking
// the origin/HEAD symbolic ref first, then falling back to main / master
// for repos without a configured remote HEAD.
func defaultBranch(repoPath string) string {
	if out, err := exec.Command("git", "-C", repoPath, "symbolic-ref", "--short", "refs/remotes/origin/HEAD").Output(); err == nil {
		return strings.TrimPrefix(strings.TrimSpace(string(out)), "origin/")
	}
	for _, b := range []string{"main", "master"} {
		if exec.Command("git", "-C", repoPath, "rev-parse", "--verify", "--quiet", "refs/heads/"+b).Run() == nil {
			return b
		}
	}
	return ""
}

type trackInfo struct {
	gone  bool
	ahead int
}

// branchTracking returns upstream tracking metadata per local branch in one
// `git for-each-ref` call. `[gone]` flags a deleted upstream; `[ahead N]`
// flags local commits not pushed.
func branchTracking(repoPath string) map[string]trackInfo {
	out, err := exec.Command("git", "-C", repoPath, "for-each-ref", "--format=%(refname:short) %(upstream:track)", "refs/heads/").Output()
	if err != nil {
		return nil
	}
	info := make(map[string]trackInfo)
	for line := range gitLines(out) {
		name, track, ok := strings.Cut(line, " ")
		if !ok || name == "" {
			continue
		}
		var ti trackInfo
		if strings.Contains(track, "[gone]") {
			ti.gone = true
		}
		if _, rest, ok := strings.Cut(track, "ahead "); ok {
			end := 0
			for end < len(rest) && rest[end] >= '0' && rest[end] <= '9' {
				end++
			}
			if end > 0 {
				ti.ahead, _ = strconv.Atoi(rest[:end])
			}
		}
		info[name] = ti
	}
	return info
}

func isDirty(wtPath string) bool {
	out, err := exec.Command("git", "-C", wtPath, "status", "--porcelain").Output()
	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(out))) > 0
}
