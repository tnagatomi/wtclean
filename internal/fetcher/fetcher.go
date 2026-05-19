// Package fetcher runs `git fetch` against a single repository on
// demand from the TUI. It wraps a single shell-out so the TUI does not
// need to know about exec.Command directly.
package fetcher

import (
	"fmt"
	"os/exec"
)

// Fetch runs `git -C <repoPath> fetch` and returns nil on success or an
// error including the captured git stderr on failure. Callers run this
// off the main goroutine (e.g. inside a tea.Cmd) since network fetches
// can take seconds.
func Fetch(repoPath string) error {
	full := []string{"-C", repoPath, "fetch"}
	out, err := exec.Command("git", full...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, out)
	}
	return nil
}
