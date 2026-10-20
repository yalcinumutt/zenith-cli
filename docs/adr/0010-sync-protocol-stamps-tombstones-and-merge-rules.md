---
status: accepted
---

# Sync protocol: sidecar field stamps, tombstones with delete-wins, merge-by-name tags

Settles the details left open by the folder-sync research. Transport is the per-device op-log in a user folder; merge is per-field last-writer-wins ordered by `(hlc, device_id)` (ADR 0008 retired cr-sqlite).

- **Stamps** live in a sidecar `field_versions(entity, uuid, field, hlc, device_id)` table, written by the core in the same transaction as the mutation (ADR 0002). Entity tables stay free of per-column stamp columns, and one generic apply function serves every entity.
- **Deletes** are a stamped `deleted_at` field, never a `DELETE` on a synced table. A delete wins over a concurrent edit of any other field; coming back requires an explicit restore (`deleted_at = null` with a newer stamp). Tombstones are kept forever.
- **Cascades** are explicit: the core writes child tombstones when a Project, Heading or Task is deleted. A live row whose container is tombstoned is read as container-less, and the core later repairs it by writing `project = null`.
- **Synced**: Areas, Projects, Headings, Tasks (Subtasks included), Repeaters, Tags, `task_tags` (stamped add/remove membership), Habits, `habit_logs`, finished time logs, `task_history` (the last three grow-only, keyed by uuid). **Device-local**: reminders and the running time log (a time log becomes synced when stopped); `sync_state`, `deferred_ops` and peer cursors.
- **Tags**: the `UNIQUE` on `tags.name` is dropped. The core finds-or-creates by case-insensitive name locally; on apply, two live tags with the same normalized name resolve to the lowest uuid, the other tombstoned with `merged_into` and its `task_tags` repointed. Every device reaches the same winner without coordination.
- **Ordering**: ops that reference a uuid not yet present are parked in `deferred_ops` and retried on each import.
- **Integrity**: `PRAGMA foreign_keys` stays OFF; the dead `ON DELETE` clauses are dropped when tables are rebuilt; the core owns referential integrity.
- **Forward compatibility**: every op-log segment header carries a `schema_version`. A device that sees a segment newer than it understands does not apply it and does not advance its cursor for that peer, then retries after it is upgraded, so an old device never silently drops fields it cannot store.
- **Clocks**: the HLC and the device uuid live in a device-local `sync_state` row, assigned inside the core's transaction wrapper. A remote stamp more than 24 hours ahead of local time is held and retried, with a clock-skew warning in the UI, instead of being applied.

## Considered options

- **Sibling stamp columns or one stamp per row:** doubles every table's columns and migrations, or loses concurrent edits to different fields of one row.
- **Edit resurrects a deleted row:** friendlier to the editing device, but a deleted task can silently reappear.
- **Name-derived uuid for tags:** breaks when a tag is renamed and the old name is created again on another device.
- **Syncing reminders across devices:** needs a rule for which device fires; deferred as a cheap follow-up.

## Consequences

- Every mutation goes through the core's transaction wrapper, which writes the row and its field stamps together (and the outgoing op, once the folder exchange ships; see ADR 0011).
- Deleted rows accumulate; compaction is a post-v1 problem.
- A reminder fires only on the device where it was set.
