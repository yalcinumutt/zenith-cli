# Two-ID row identity: integer handle plus sync UUID

v1 needs a sync identity (a client-generated, globally stable id; auto-increment integers collide across offline devices) but the CLI's UX depends on short typed integers (`zenith task done 5`). We keep `id INTEGER PRIMARY KEY` as-is for every table — local handle, FKs, CLI args all unchanged — and add a `uuid` column as the sync identity, added to every entity table (not just tasks) in this same migration so the shape is right before any sync code depends on it. The alternative, a UUID primary key with the CLI falling back to per-listing display indices, was rejected: it means the same task has a different integer on every device, which is worse UX than today, and it forces a full table-rebuild migration now for a UX cost that isn't worth paying.

Trade-off accepted: sync code must translate between integer FKs and UUIDs at the op-log boundary, rather than having one identity throughout.
