# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
go build -o zenith main.go          # build the binary (gitignored)
go run main.go <command>            # run without building
go test ./...                       # all tests (internal/storage and cmd have tests)
go test ./internal/storage -run TestSQLiteStore_TasksWithTagsAndTimers   # single test
```

There is no Makefile or linter config. Releases are cut by pushing a `v*` tag, which triggers `.github/workflows/release.yml` (GoReleaser, `.goreleaser.yaml`). The CLI version string is hardcoded in `cmd/root.go` (`Version`), and `zenith update` self-updates from GitHub releases (`go-github-selfupdate`), so archive names in `.goreleaser.yaml` must stay compatible with it.

## Architecture

Go CLI (module `github.com/yalcinumutt/zenith-cli`) for tasks, habits, projects and time tracking. Cobra commands + Bubble Tea TUI + SQLite.

- `main.go` → `cmd.Execute()`. Each file in `cmd/` registers one cobra command tree (task, habit, project, plan, summary, log, search, timer, ai, tui, update).
- `cmd/root.go` holds the package-level `store storage.Store`, initialized in `PersistentPreRunE` for **every** command (opens the DB, runs schema + migrations). It also runs `checkExpiredTimersPassive()`, which fires macOS notifications (`osascript`) for due scheduled timers on any invocation; this is skipped for the `daemon` command (`cmd/timer.go`), which polls actively.
- `internal/storage`: `Store` interface (`store.go`) with the single `SQLiteStore` implementation (mattn/go-sqlite3, needs CGO to build locally). DB lives at `~/.zenith/data.db`; tests use `NewSQLiteStoreAtPath` with a temp file. Schema is `CREATE TABLE IF NOT EXISTS` in `sqlite_store.go`; **schema changes to existing tables go in `migrate.go`** as idempotent steps guarded by `columnExists`, since existing user DBs must upgrade in place. A default "Anytime" project is ensured on open. Adding a storage method means updating the `Store` interface and `SQLiteStore`.
- `internal/models`: plain structs shared by storage, cmd and TUI.
- `internal/tui/app.go`: Bubble Tea dashboard (`zenith tui`); `internal/ui/styles.go`: shared Lipgloss styles used by cmd output.
- `internal/ai`: talks to a local OpenAI-compatible endpoint (Ollama or LM Studio), configured via env vars `ZENITH_AI_PROVIDER` (`ollama` default / `lmstudio`), `ZENITH_AI_URL`, `ZENITH_AI_MODEL`. Used by `cmd/ai.go` for task suggestions and summaries.
- `website/`: static marketing/docs site (plain HTML/CSS/JS), independent of the Go build. `dist/` is GoReleaser output.

Note: CI and the release workflow read the Go version from `go.mod`; README/CONTRIBUTING may mention older versions, so trust `go.mod`.

## Agent skills

### Issue tracker

Issues live in GitHub Issues (`gh` CLI). See `docs/agents/issue-tracker.md`.

### Triage labels

Default five-label vocabulary (`needs-triage`, `needs-info`, `ready-for-agent`, `ready-for-human`, `wontfix`). See `docs/agents/triage-labels.md`.

### Domain docs

Single-context: one `GLOSSARY.md` + `docs/adr/` at the repo root. See `docs/agents/domain.md`.
