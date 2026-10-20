---
status: accepted
---

# Use the pure-Go SQLite driver (modernc); sync is hand-rolled, not cr-sqlite

The shared core, CLI, TUI and desktop app open the same SQLite file through `modernc.org/sqlite` instead of `mattn/go-sqlite3`. mattn needs CGO and a C toolchain on every OS, and the CLI's GoReleaser config already builds with `CGO_ENABLED=0`, which compiles mattn to a stub that fails on first use, so the published CLI cannot open its database. modernc keeps the CLI and TUI CGO-free (single-runner cross builds, a Windows target, no toolchain for contributors). The desktop app still needs CGO on macOS and Linux for the webview; that is the webview's cost, not the database's.

A spike confirmed the swap is low-risk: the storage tests pass with `CGO_ENABLED=0`, and CLI output against a copy of a real database was identical between the two drivers. The file format is plain SQLite, so existing databases open unchanged.

modernc cannot load native extensions, so cr-sqlite is no longer an option. Sync is the hand-rolled per-field last-writer-wins design (HLC plus device id, tombstones) from the folder-sync research; the data model does not need to stay cr-sqlite-compatible.

## Considered options

- **Stay on mattn, build with `CGO_ENABLED=1`:** keeps extension loading (and so cr-sqlite), but needs per-OS runners with C toolchains for every artifact and makes Windows cross-builds depend on Docker.

## Consequences

- Connection setup lives in one `openDB` in `internal/storage`: driver name `"sqlite"`, pragmas passed in the DSN so every pooled connection gets them (`journal_mode=WAL`, `busy_timeout=5000`, `synchronous=NORMAL`). `foreign_keys` is decided by the sync-protocol ticket.
- The release binary grows by about 2 MB and SQLite is slower than the C build; neither matters for a task list.
- Evidence: spike on modernc `v1.60.1`, October 2026; folder-sync research on branch `research/folder-sync`.
