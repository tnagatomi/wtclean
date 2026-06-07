# wtclean

A TUI tool for listing and deleting stale git worktrees across many repositories.

If you spread worktrees across several projects, it's easy to lose track of which
ones are merged, abandoned, or safe to delete. `wtclean` scans your configured
roots, gathers every repository's worktrees, and labels each with **badges** so
you can see at a glance what is disposable — then delete in bulk, safely.

## Features

- Scans multiple root directories for git repositories and their worktrees.
- Badges each worktree with its state (`merged`, `upstream-gone`, `uncommitted`,
  `unpushed`, `locked`, `no-dir`, `primary`).
- One key selects every **safe-to-remove** worktree — clean, disposable, and
  carrying no warning state.
- Optionally deletes the associated branches alongside the worktrees.
- Incremental filter, emacs-style movement keys, and a built-in keyboard
  reference (`?`).

## Install

### Homebrew

```sh
brew install tnagatomi/tap/wtclean
```

### go install

```sh
go install github.com/tnagatomi/wtclean/cmd/wtclean@latest
```

## Getting started

1. Generate a starter config:

   ```sh
   wtclean init
   ```

   This writes `~/.config/wtclean/config.yml` (or `$XDG_CONFIG_HOME/wtclean/config.yml`).

2. Edit it to point at the directories that hold your repositories:

   ```yaml
   # Root directories to scan for git repositories.
   # Each root is walked recursively. Tilde (~) expands to your home directory.
   roots:
     - ~/src
     - ~/work

   # Maximum recursion depth from each root.
   max_depth: 5
   ```

3. Launch the TUI:

   ```sh
   wtclean
   ```

## Usage

`wtclean` is a three-screen flow:

1. **Repository list** — every repository found under your roots. `enter` opens one.
2. **Worktree list** — the selected repository's worktrees, each with badges.
   Toggle individual rows, or press `s` to select all safe-to-remove worktrees.
3. **Delete confirmation** — review the selection, optionally also delete the
   branches, and confirm.

### Key bindings

Press `?` at any time for the full reference. The essentials:

| Key            | Action                                          |
| -------------- | ----------------------------------------------- |
| `↑`/`k`, `↓`/`j` | Move the cursor                               |
| `enter`        | Open the focused repository                      |
| `space`        | Toggle selection on the focused worktree         |
| `s`            | Select all safe-to-remove worktrees              |
| `/`            | Incremental filter                               |
| `d`            | Open the delete confirmation                      |
| `r`            | Refresh (repo list) / fetch and reload (worktree list) |
| `esc`          | Clear filter, or go back one screen              |
| `q`, `ctrl+c`  | Quit                                             |

### Badges

**Safe to remove** — positive evidence a worktree is disposable:

- `merged` — the branch is merged into the repository's default branch (a local
  `git branch --merged` check).
- `upstream-gone` — the branch's upstream tracking branch was deleted, usually a
  squash- or rebase-merged PR whose head was auto-deleted.
- `no-dir` — the working directory is already gone; only an administrative record remains.

**Removal loses local work** — warning states that block safe selection:

- `uncommitted` — has changes not yet committed.
- `unpushed` — has commits not yet pushed.
- `locked` — deliberately protected with a worktree lock.
- `primary` — the main checkout of a repository; never selectable or deletable.

A worktree is **safe-to-remove** only when it carries at least one positive badge
and none of the warning states. A clean, pushed, but unmerged worktree is *not*
safe-to-remove: the absence of warnings is not evidence that it is disposable.

> **Refresh vs. fetch:** `r` on the repository list rescans your roots locally and
> never touches the network. `r` on the worktree list fetches the current
> repository from its remote, updating remote-derived badges such as `upstream-gone`.

## Configuration

| Field       | Description                                                      | Default |
| ----------- | --------------------------------------------------------------- | ------- |
| `roots`     | Root directories to scan, walked recursively. `~` expands to home. | —    |
| `max_depth` | Maximum recursion depth from each root.                         | `5`     |

The config file lives at `$XDG_CONFIG_HOME/wtclean/config.yml`, falling back to
`~/.config/wtclean/config.yml`.

## License

[MIT](LICENSE)
