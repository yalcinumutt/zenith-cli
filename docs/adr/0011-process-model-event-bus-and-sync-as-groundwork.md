---
status: accepted
---

# Process model: one core `Tick`, a small event bus, and sync shipped as groundwork

- **Ticking.** The core exposes a single `Tick(now)` that claims due reminders atomically (ADR 0002) and delivers them through the injected `Notifier`. Each process decides when to call it: the desktop app on an in-process ticker while it runs, `zenith daemon` as the headless equivalent, and every CLI invocation as today's passive check (now through the Notifier, not `osascript`). In v1, closing the desktop window quits the app on all three OSes, so a reminder fires only while the app or the daemon runs. Hide-to-tray, autostart and background running are post-v1.
- **Events.** The core has an in-process `Subscribe(func(Event)) (unsubscribe)`. Public events in v1: `TaskChanged{uuid, kind}`, `ListsInvalidated{keys}`, `ReminderDue{uuid}`, and `ExternalChange` (a bare "changed outside this process", produced by the shell's `data_version` poll). Payloads carry ids and a coarse kind, never whole entities. The Wails shell forwards them to the UI's query invalidation; a future tray, plugin host or loopback API subscribes to the same bus. New events are additive.
- **Calendar.** v1's When stays date-only. Time of day and duration for timeblocking arrive later as new nullable, separately stamped fields; nothing is added to the v1 model for it.
- **Sync scope.** v1 ships the groundwork only: uuids (ADR 0003), per-field stamps and soft deletes on every mutation (ADR 0010), and the device id and HLC. The folder exchange (writing and reading segments, the apply loop, a "sync folder" setting) is the first feature after v1. Stamping from the first mutation avoids backfilling invented stamps later.

## Considered options

- **Full sync in v1:** a bigger beta on top of an already risky Wails v3 bet, with cross-device conflict testing in the first release.
- **Background running in v1 (tray, hide on close):** best-effort on Linux and needs v3 tray features; deferred.

## Consequences

- Release notes and README must say that reminders only fire while the app or `zenith daemon` is running.
- Sync-worker single-writer rule (one process per device writing the folder, for example a lease row) is not decided here: no v1 process writes the folder. It is an open item for the first post-v1 sync ticket.
- Op-log segment headers carry a `schema_version`; see ADR 0010.
