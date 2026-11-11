---
status: accepted
---

# The desktop adapter addresses entities by integer handle and carries the uuid alongside

ADR 0002 described the Wails adapter speaking "string UUIDs". In practice the adapter (`internal/appapi`) addresses every entity by its **integer handle**, the same id the CLI and TUI use, and returns the **uuid** as an additional field on every DTO. Calls take `id: number`; results carry `id` and `uuid`.

Why: the handle is already the local, short, stable address everywhere else (ADR 0003), so the desktop app, CLI and TUI can all refer to the same task by the same number, and the core's API is keyed by handle. Resolving uuid to handle at the boundary would add a lookup to every call for no local benefit. JavaScript numbers hold the handles exactly (far below 2^53).

The uuid still crosses the boundary on every DTO, so nothing the front end sees prevents using it as the identity later (for example if a future front end holds entities across devices). Sync translates between handles and uuids at the op-log boundary in the core (ADR 0003, ADR 0010), not in the adapter.

## Consequences

- DTOs carry plain values only: dates are `YYYY-MM-DD` strings, instants are RFC 3339 strings, no core types or `time.Time`.
- The adapter holds no framework imports, so it is tested without a window.
