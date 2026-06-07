# Async startup scan inside the TUI

The repository list is now produced by an async `repo.Discover` launched from
the bubbletea model's `Init()`, sharing one code path with the `r` (refresh)
action, rather than being computed synchronously in `runTUI` before the program
starts. We did this so refresh and startup share a single load path and so the
UI appears immediately with a `⏳ Scanning...` indicator instead of a blank
terminal while the filesystem is scanned.

## Considered Options

- **Synchronous discover in `runTUI` (previous behavior)** — kept the clean
  non-zero exit on a discover failure (friendly to scripting), but forced two
  separate paths for getting repos into the model and showed nothing until the
  scan finished.
- **Async discover in `Init()` (chosen)** — unifies startup and refresh.

## Consequences

- A `Discover` failure is now surfaced inside the TUI (`⚠ scan failed`) with
  `r` to retry, and the process exits 0 on normal quit — it no longer produces
  a pre-TUI stderr message and non-zero exit code. Scanner failures are rare
  enough that this trade is acceptable.
- `config.Load` stays synchronous and pre-TUI: a missing or invalid config
  leaves nothing useful to show, so it still aborts before the program starts
  with a proper exit code.
