---
status: accepted
---

# Business rules live in a transport-agnostic core; the store is plain persistence

Today `storage.Store` is the only seam, so rules leak into the SQLite store (history entries, "starting a timer stops the running one") and into `cmd/` (find-task-by-ID loops, reminder delivery with `osascript`). We introduce `internal/core` with one service per aggregate (tasks, projects, habits, reminders), built together by `core.New(store, clock, notifier)`. Services own status transitions, history, timer exclusivity, recurrence and "not found" errors; the store keeps CRUD plus a `WithTx` helper. Core imports no Cobra, Bubble Tea, Lipgloss or Wails. The CLI, TUI and Wails shell are thin front-ends over it, and sync later hooks into the core as the single place mutations happen.

## Considered options

- **One `App` facade:** simpler to hold, but grows into a god object.
- **Keep `Store` as the seam plus helpers:** least work, but rules stay duplicated per front-end and sync has no single mutation point.

## Consequences

- The Wails shell wraps core services in an adapter that speaks plain DTOs (string UUIDs, ISO dates) and generates the typed TS client; core types never carry JSON or Wails concerns.
- Reminders: `Reminders.Due(now)` claims due items atomically (single `UPDATE ... WHERE triggered=0`) so the CLI and the desktop app never both fire one. Delivery is an injected `Notifier` with macOS, Windows and Linux implementations. `daemon` stays a CLI ticker over `Due`; the desktop app runs the same tick in-process.
- The core publishes in-process change events. The desktop shell also polls `PRAGMA data_version` (SQLite in WAL mode with a busy timeout) to catch writes from the CLI or the sync importer. File watchers are rejected as unreliable on Windows and network folders.
- Sync stays folder-based; there is no HTTP sync transport. Because the core is transport-agnostic, a local loopback Chi API (for scripts or editor plugins) can be added after v1 as a fourth front-end without rework.
- No test-suite commitment for the v1 spec. The injected `Clock` and `Notifier` keep tests cheap to add later.
