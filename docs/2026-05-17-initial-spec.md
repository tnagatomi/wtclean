# wtm — Initial Specification

Date: 2026-05-17
Status: Design (pre-implementation)

## Purpose

`wtm` is a TUI tool for developers that lists and deletes git worktrees across multiple projects. The primary use case is cleaning up stale worktrees that accumulate across repositories over time.

## Stack

- **Go 1.26.3**, module path `github.com/tnagatomi/wtm`
- **bubbletea + bubbles + lipgloss** for the TUI
- **cobra** for CLI dispatch
- **goccy/go-yaml** for config parsing
- License: **MIT**

## Discovery

- The user configures one or more root directories.
- Each root is walked recursively. When a `.git` is found, descent into that subtree stops (pruning).
- Maximum recursion depth defaults to 5.
- Detection rules:
  - `.git` directory → treat as a repository (target)
  - `.git` file (linked worktree marker) → skip; the parent repo will enumerate it via `git worktree list`
  - Bare repositories → target (they are common worktree hosts)
- Symbolic links are not followed (loop avoidance).
- Inaccessible directories are silently skipped and logged.

## Configuration

- Location: `~/.config/wtm/config.yml` (XDG-compliant).
- A `wtm init` subcommand generates a starter file.
- Schema (MVP):

```yaml
roots:
  - ~/ghq
  - ~/work
max_depth: 5
```

Excluded from MVP: exclude patterns, glob roots, sort keys, color themes, custom default branch names.

## UI Structure

The TUI is a two-screen drill-down: a repository list, then a worktree list per repository.

### Screen 1 — Repository List

Columns:
- Full path of the repository
- Linked worktree count (primary excluded from the count)
- Last fetch time, formatted `yyyy-mm-dd hh:mm:ss`, derived from `.git/FETCH_HEAD` mtime

Behavior:
- Repositories with zero linked worktrees are hidden.
- Sort order: repository path, alphabetical.
- Keys:
  - `enter` — open the focused repo (go to Screen 2)
  - `q` — quit
  - `?` — help
  - `j` / `k` / `↑` / `↓` — move cursor
  - `g` / `G` / `home` / `end` — jump to top / bottom
- `r` is intentionally not bound here, to keep its meaning consistent with Screen 2.

### Screen 2 — Worktree List (within one repository)

Columns:
- Selection checkbox
- Full path of the worktree (no abbreviation, no tilde shortening)
- Branch name
- Last commit time, `yyyy-mm-dd hh:mm:ss`
- Badges

Badges:
- `[primary]` — the primary checkout; cannot be selected
- `[merged]` — branch merged into the default branch
- `[gone]` — upstream tracking branch is gone
- `[dirty]` — uncommitted changes present
- `[unpushed]` — local commits not pushed
- `[locked]` — `git worktree lock` is set
- `[missing]` — directory has been removed manually; only an administrative record remains

Behavior:
- Sort order: last commit time, oldest first.
- The primary worktree is shown but not selectable.
- Filter:
  - `/` enters an incremental filter
  - Matching is case-insensitive substring
  - Targets: path and branch name only (badge names are not searchable in MVP)
- Keys:
  - `space` — toggle selection on the focused row
  - `/` — start filter, `esc` — clear filter
  - `d` — open the delete confirmation screen
  - `r` — run `git fetch` for this repo and reload
  - `esc` — back to Screen 1
  - `q` — quit
  - `?` — help

### Delete Confirmation Screen

A single screen confirms the entire batch — there is no per-item confirmation.

Layout:

```
Deleting 3 worktrees:

    /Users/.../wtm/wt/feat-x      [merged]
  ⚠ /Users/.../api/wt/wip         [dirty][unpushed]
  ⚠ /Users/.../infra/wt/exp       [locked]

Options:
  [x] Also delete branches

⚠ Warnings (deletion will be forced):
  - dirty:    uncommitted changes will be lost (1)
  - unpushed: commits not pushed will be lost (1)
  - locked:   the lock will be released (1)

[y] Confirm    [n] Cancel
```

Rules:
- "Also delete branches" is an opt-in toggle.
  - Default: ON only when every selected worktree carries the `[merged]` badge.
  - Default: OFF otherwise.
- Warning block is informational only; the deletion proceeds regardless when `y` is pressed.
- Single confirmation key: `y` runs the entire batch; `n` cancels.

## Delete Semantics

- Worktree removal: `git worktree remove <path>`, with `--force` when dirty/unpushed/locked.
- Locked worktree: `git worktree unlock` first, then force remove.
- Missing worktree: handled internally as `git worktree prune`.
- Branch deletion (toggle ON): `git branch -d` for merged branches, `git branch -D` for unmerged.
- The primary worktree is never deletable.

## Badge Computation

- Computed entirely from local refs. The tool does not automatically run `git fetch`.
- Staleness of `[merged]` / `[gone]` is the user's responsibility: Screen 1 exposes the last fetch time per repo, and Screen 2 exposes `r` to fetch the current repo on demand.

## Error Handling

- Continue-on-error across all operations (scan, fetch, remove, prune).
- One-line summary per failure is shown in the TUI results area.
- Full details are appended to `~/.local/state/wtm/wtm.log` (XDG_STATE_HOME).
- Startup scan shows a `scanning N/M repos` progress indicator; delete and fetch show a spinner.

## CLI Surface

- `wtm` (no arguments) — launch the TUI
- `wtm init` — write a starter `config.toml`
- `wtm --version` / `-v`
- `wtm --help` / `-h`

Reserved for later phases: `wtm doctor`, `wtm list --json`, `wtm rm`, `--config <path>`.

## Color and Styling

- lipgloss adaptive color (auto-selects for dark/light terminals).
- Badge palette:
  - `[merged]` — green
  - `[gone]` — yellow
  - `[dirty]`, `[unpushed]` — red
  - `[locked]` — purple
  - `[primary]`, `[missing]` — gray
- No theme switcher in MVP.

## Empty States

| Situation | Message |
| --- | --- |
| Config file missing | Error: run `wtm init` to generate a starter file. |
| Config file has zero roots | Error: edit `config.toml` to add at least one root. |
| No repositories found under any root | `No repositories found under: ...` plus the config path. |
| All repositories filtered out (only primaries exist) | `No worktrees found. (Repositories with only primary checkouts are hidden.)` |

## Distribution

- `go install github.com/tnagatomi/wtm@latest`
- GoReleaser pushes per-OS binaries to GitHub Releases on tag.
- Homebrew tap: existing `tnagatomi/homebrew-tap` receives a new `Formula/wtm.rb`. Install via `brew install tnagatomi/tap/wtm`.

## CI and Release Authentication

- GitHub Actions runs tests on pull requests and the release pipeline on tag push.
- Cross-repo write (wtm → homebrew-tap) is authenticated with a **GitHub App installation token**, minted at workflow time via `actions/create-github-app-token@v1`.
- No personal access token is used.

Workflow sketch:

```yaml
- uses: actions/create-github-app-token@v1
  id: app-token
  with:
    app-id: ${{ secrets.APP_ID }}
    private-key: ${{ secrets.APP_PRIVATE_KEY }}
    owner: tnagatomi
    repositories: wtm,homebrew-tap
- uses: goreleaser/goreleaser-action@v6
  env:
    GITHUB_TOKEN: ${{ steps.app-token.outputs.token }}
```

## Testing Strategy

- Unit tests: config parsing, `git worktree list --porcelain` parsing, root scanner (fed with synthetic `.git` directory trees in tmp dirs).
- Integration tests: real `git` invocations against tmp repositories for remove / prune / fetch flows.
- TUI tests: bubbletea `teatest` with golden files, covering startup, selection, confirmation, and cancel.

## Out of Scope for MVP (Phase 2 Candidates)

- Flat (single-screen) view and grouped views across projects
- Additional CLI subcommands (`list --json`, `rm`, `doctor`, `--config <path>`)
- Bulk refresh of all repositories
- Sort key switching and alternative orderings
- Aggregate badges on Screen 1
- Exclude patterns and glob root specifications in config
- Color theme switching
- Linux package distribution (apt / snap)
