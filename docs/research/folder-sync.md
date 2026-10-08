# Local-first sync over a user-provided folder with SQLite

Research for wayfinder issue #4 (child of map #2). Researched 2026-10-08. Primary sources only; each claim cites its source. Statements marked **[inference]** are my own reasoning from the cited facts, not something a source says. Statements marked **[unverified]** I could not confirm from a primary source.

## Answer in brief

1. **Do not put the live `~/.zenith/data.db` in the synced folder.** SQLite's own docs say a database file must travel together with its journal/WAL, that copying it mid-transaction yields a corrupt copy, and that network filesystem locking is unreliable. Folder-sync tools copy files independently, with no regard for any of that.
2. **Keep the live DB local. Exchange changes through the folder as per-device, single-writer, append-only files** (an op-log of immutable segment files). Every provider's conflict machinery only fires when two devices write the *same file*; one writer per file avoids it entirely **[inference]**.
3. **Merge rule: last-writer-wins per field, ordered by a Hybrid Logical Clock plus device id, with tombstones for deletes.** Hand-rolled LWW is a small amount of code for a task-list data model. cr-sqlite is the off-the-shelf alternative but imposes schema constraints Zenith's current schema violates and adds a native-extension build burden on three OSes. Neither choice removes the need for the folder transport (cr-sqlite produces change rows; you still ship them).
4. **Every synced row needs a globally unique, stable, client-generated ID** (UUID; v7 recommended). Auto-increment `int64` ids collide across devices.
5. **Per-provider differences are real** (Google Drive has no Linux client; iCloud may evict files; Dropbox/Syncthing mint conflict copies). The log design should tolerate: missing files, late-arriving files, torn/partial files, duplicate delivery, and stray `conflict` copies.

## 1. Why syncing the `.db` file directly is unsafe

### 1.1 The database is more than one file

- WAL mode adds a `-wal` file and a `-shm` shared-memory file next to the database. ([SQLite WAL](https://www.sqlite.org/wal.html), section 1 item 6)
- "The WAL file is part of the persistent state of the database and should be kept with the database if the database is copied or moved. If a database file is separated from its WAL file, then transactions that were previously committed to the database might be lost, or the database file might become corrupt." ([SQLite WAL](https://www.sqlite.org/wal.html), section 4)
- Rollback-journal mode has the same hazard: "Copying a database file without also copying its journal" and "Overwriting a database file with another without also deleting any hot journal associated with the original database" are listed as corruption causes. ([How To Corrupt An SQLite Database File](https://www.sqlite.org/howtocorrupt.html), sections 1.3-1.4)
- Zenith does not set WAL today (`sql.Open("sqlite3", dbPath)` with no pragma, `internal/storage/sqlite_store.go:40`), so it runs in the default rollback-journal mode. Either way the file set is not a single file. A sync client uploads files independently and not as an atomic set **[inference]**; the sources above say what happens when the set is separated.

### 1.2 Copying a live database yields corruption

- "Systems that run automatic backups in the background might try to make a backup copy of an SQLite database file while it is in the middle of a transaction. The backup copy then might contain some old and some new content, and thus be corrupt." ([howtocorrupt](https://www.sqlite.org/howtocorrupt.html), section 1.2)
- Safe copy methods per SQLite: `sqlite3_rsync`, `VACUUM INTO`, or the backup API. ([howtocorrupt](https://www.sqlite.org/howtocorrupt.html), section 1.2) None of these is what a sync client does.
- Consequence for Zenith: the CLI is invoked frequently and short-lived, and `checkExpiredTimersPassive()` plus a `daemon` also write (see `CLAUDE.md`). A sync client can therefore observe the file mid-transaction **[inference]**.

### 1.3 Network and cloud filesystems break SQLite's locking assumptions

- "SQLite depends on the underlying filesystem to do locking as the documentation says it will. But some filesystems contain bugs in their locking logic such that the locks do not always behave as advertised. This is especially true of network filesystems..." ([howtocorrupt](https://www.sqlite.org/howtocorrupt.html), section 2.1)
- "WAL does not work over a network filesystem" because it needs shared memory between processes on one host. ([SQLite WAL](https://www.sqlite.org/wal.html), section 1 item 1 and section 2.2)
- SQLite's page on network use says remote-database use is "at the user's risk," the library is not tested across a network, and recommends a client/server DB when data and application are separated. ([SQLite Over a Network](https://www.sqlite.org/useovernet.html))
- Caveat: cloud-sync folders are not network filesystems in the NFS sense (the file is a local replica). These sources address NFS-style mounts, not iCloud/Dropbox replicas. The relevant hazards for replicas are 1.1, 1.2 and the whole-file conflict model below **[inference]**. Streamed/on-demand modes (Google Drive "stream", iCloud eviction) are closer to remote files.

### 1.4 Providers resolve conflicts at whole-file granularity, with no merge

- Dropbox: a conflicted copy is created when the same file is changed in two places before sync (two users at once, one editing offline, or a file left open elsewhere and auto-saved). The conflicted copy is named with the editor, "conflicted copy" and the date, and "the last version saved will always appear as the conflicted copy." Dropbox's advice is to compare and merge manually. ([Dropbox: What's a conflicted copy?](https://help.dropbox.com/organize/conflicted-copy))
- Syncthing: on concurrent modification with differing content, the older-mtime file is renamed `<filename>.sync-conflict-<date>-<time>-<modifiedBy>.<ext>`; ties go to the larger device-ID value; conflict copies "are treated as normal files after they are created, so they are propagated between devices." ([Syncthing: Conflicting Changes](https://docs.syncthing.net/users/syncing.html))
- iCloud Drive: the most recently modified file becomes current and others become conflict versions; non-document-scope conflicts are resolved automatically by "picking up a winning version and keeping the other changes in a losing version"; same-name new files in the Documents folder get a numeric "bounced" suffix. Unresolved versions sync to all peers and consume quota. ([Apple TN2336](https://developer.apple.com/library/archive/technotes/tn2336/_index.html); [Apple Support: document version conflicts](https://support.apple.com/guide/mac-help/mh40780/mac))
- Net: the loser of a conflict is a whole database file. Row-level merge of two SQLite files is not offered by any of them. The Ink & Switch essay makes the same point generically: when a file is edited on two devices "the result is a conflict that needs to be merged manually." ([Local-first software](https://www.inkandswitch.com/essay/local-first/))

## 2. Candidate designs

### A. Sync the live `.db` (rejected)
See section 1. Not recommended.

### B. Per-device database files, merged on read
Each device periodically exports a consistent snapshot (`VACUUM INTO` is a documented safe copy method: [howtocorrupt 1.2](https://www.sqlite.org/howtocorrupt.html)) to `device-<id>.db` in the folder; other devices read peers' snapshots and merge. Single writer per file avoids provider conflicts **[inference]**. Costs: re-uploads the whole DB on every change, and you still need the full merge algorithm and row identity. Only a fallback.

### C. Per-device op-log (recommended transport)
Each device writes its own journal into the folder, e.g. `ops/<device-id>/<seq>-<hlc>.seg`, never touching another device's files. Properties:

- **Single writer per file**: the conflict mechanisms in 1.4 only trigger when two devices modify one file, so none fire **[inference]**.
- **Immutable segments (write temp, then rename)** are preferable to appending to one growing file. Syncthing itself writes via a hidden temp file and replaces on completion ("never writes directly to a destination file"; temp files `.syncthing.<name>.tmp` / `~syncthing~<name>.tmp` are ignored by Syncthing: [Syncthing syncing docs](https://docs.syncthing.net/users/syncing.html)), but no source I found promises other providers deliver a file atomically, so readers must tolerate a partial file: length-prefixed records plus a checksum, ignore a bad tail, retry later **[inference; atomic delivery for Dropbox/iCloud/Drive is [unverified]]**.
- **Idempotent, order-tolerant apply**: segments can arrive late, twice, or out of order across devices; per-field LWW with a total order (section 3) makes re-applying harmless **[inference]**.
- **Per-peer cursor**: each device stores, locally, the highest segment it has applied from each peer (this is what cr-sqlite does with `crsql_tracked_peers`, keyed by `site_id`: [bootstrap.rs](https://github.com/vlcn-io/cr-sqlite/blob/main/core/rs/core/src/bootstrap.rs)).
- **Bootstrap/compaction**: a new device needs either all segments from the start or a periodic per-device snapshot checkpoint. Compaction is safe only for segments every known device has passed **[inference]**; the folder has no membership service, so "every known device" must itself be tracked (e.g. devices write a `seen` marker).

### D. Merge semantics: CRDT options

**D1. cr-sqlite** ([repo](https://github.com/vlcn-io/cr-sqlite), MIT)
- What it is: a "run-time loadable extension for SQLite" adding multi-master replication via CRDTs; tables become CRRs via `SELECT crsql_as_crr('table')`; changes are read from and applied via the `crsql_changes` virtual table with columns `table, pk, cid, val, col_version, db_version, site_id, cl, seq`. ([README](https://github.com/vlcn-io/cr-sqlite))
- Conflict rule (read from source): higher `col_version` wins; if equal, compare values; if equal values and the option is on, tie-break on `site_id`. ([changes_vtab_write.rs](https://github.com/vlcn-io/cr-sqlite/blob/main/core/rs/core/src/changes_vtab_write.rs)) This is per-column LWW using a logical (counter) clock, with causal length `cl` for row existence (deletes/resurrection). ([README](https://github.com/vlcn-io/cr-sqlite) names CLSet.)
- Schema constraints (from the source `is_table_compatible` in [tableinfo.rs](https://github.com/vlcn-io/cr-sqlite/blob/main/core/rs/core/src/tableinfo.rs) and the [constraints doc](https://vlcn.io/docs/cr-sqlite/constraints)):
  - Must have a non-nullable primary key.
  - No unique indices other than the primary key.
  - A `NOT NULL` column must have a default.
  - Foreign keys allowed but "the constraint cannot be checked."
  - Schema changes require `crsql_begin_alter` / `crsql_commit_alter`. ([README](https://github.com/vlcn-io/cr-sqlite))
- Fit against Zenith's schema (`internal/storage/schema.sql`): `tags.name TEXT NOT NULL UNIQUE` violates the unique-index rule; every table uses `INTEGER PRIMARY KEY AUTOINCREMENT` ids, which collide across devices (section 5); FKs with `ON DELETE CASCADE`/`SET NULL` would not be enforced across merges.
- Operational cost: ships as a native library loaded on every connection ("must be loaded on every connection"), built with Rust nightly; prebuilt binaries exist for macOS (arm64, x86_64), Linux (x86_64, aarch64), Windows (x86_64, i686), Android and iOS. ([README](https://github.com/vlcn-io/cr-sqlite); [latest release v0.16.3](https://github.com/vlcn-io/cr-sqlite/releases)) `mattn/go-sqlite3` supports extension loading by default (build tag `sqlite_omit_load_extension` disables it: [go-sqlite3 README](https://github.com/mattn/go-sqlite3)), so loading is feasible, but the Wails, CLI and TUI builds would each need to locate and ship the right binary per OS/arch.
- Maintenance signal (GitHub API/repo, checked 2026-10-08): not archived; last push 2026-08-10; but the latest tagged release is v0.16.3 from 2024-01-17, and the README warns "main may not be 100% stable" and to build from a tag. Inserts into CRRs are about 2.5x slower than regular tables. ([README](https://github.com/vlcn-io/cr-sqlite))
- Does not solve transport: it yields change rows; you still define how they travel through the folder (design C).
- It also keeps clock/metadata tables per CRR, so the DB format becomes cr-sqlite's, tying all three front-ends to it **[inference]**.

**D2. Hand-rolled per-field LWW (recommended)**
- Foundation: Johnson and Thomas's LWW register over "a database of registers that can be created, updated and deleted"; LWW "ensures a total order of operations, at the cost of losing concurrent updates." ([Shapiro et al., CRDTs, RR-7687](https://inria.hal.science/inria-00609399), related work)
- Granularity: stamping per field (not per row) means edits to different fields of one task (title on laptop, `when` on phone) both survive; only same-field concurrent edits lose one **[inference]**; Ink & Switch likewise says only concurrent updates to "the same property of the same object" need resolution ([Local-first software](https://www.inkandswitch.com/essay/local-first/)).
- Cost for Zenith: a small `(hlc, device_id)` stamp per synced field plus an apply function **[inference]**. No native extension, no schema restrictions beyond what you choose, and the unique `tags.name` can be handled by an explicit merge rule.
- Existing code note: `UpdateTask` rewrites every column and sets `updated_at = CURRENT_TIMESTAMP` (`internal/storage/sqlite_store.go:139`). Per-field LWW needs the store to know *which* fields changed (diff old vs new), and `CURRENT_TIMESTAMP` is a second-resolution wall clock, so it cannot serve as the sync clock.

**D3. SQLite session extension (not recommended)**
- Records changes into changesets/patchsets and applies them with a conflict handler. Requires a declared PRIMARY KEY; disabled unless built with `SQLITE_ENABLE_SESSION` and `SQLITE_ENABLE_PREUPDATE_HOOK`. ([SQLite session extension](https://www.sqlite.org/sessionintro.html))
- It detects conflicts (e.g. an UPDATE whose old values differ) but leaves resolution to your handler, so you still write the merge policy. `mattn/go-sqlite3` documents a `sqlite_preupdate_hook` build tag but I found no session-extension build tag in its README, so Go binding support is **[unverified]** and likely needs custom cgo.

### Comparison

| | Live .db in folder | Per-device snapshots | Op-log + hand LWW | Op-log + cr-sqlite |
|---|---|---|---|---|
| Provider conflicts | frequent, whole-DB | none (single writer) | none (single writer) | none (single writer) |
| Corruption risk | high (sec. 1) | low | low | low |
| Merge logic | none | you write it | you write it (small) | provided |
| Native deps | none | none | none | Rust-built loadable ext |
| Schema constraints | none | none | none | non-null PK, no non-PK unique, etc. |
| Bandwidth | full DB per change | full DB per change | deltas | deltas |

## 3. Ordering and clocks

- Lamport's logical clocks give a total order consistent with causality: events are ordered by clock value and "to break ties, we use any arbitrary total ordering" of processes. ([Lamport 1978, Time, Clocks, and the Ordering of Events](https://lamport.azurewebsites.net/pubs/time-clocks.pdf)) This is the (counter, device_id) pattern cr-sqlite uses.
- Pure logical clocks cannot be related to wall time ("it is not possible to query events in relation to physical time") and assume no side channels; wall clocks alone have uncertainty. The Hybrid Logical Clock "captures the causality relationship like logical clocks" while staying "always close to the NTP clock" and fits in 64 bits, with the property that if e happened-before f then `l.e < l.f`. ([Kulkarni et al., Logical Physical Clocks](https://cse.buffalo.edu/tech-reports/2014-04.pdf))
- Recommendation for Zenith: stamp each field write with `(hlc, device_id)`, compare lexicographically; on receive, advance the local HLC past any observed remote HLC. This preserves "later edits win" even if a device's wall clock is somewhat behind, because receiving a peer's larger stamp pushes the local clock forward **[inference]**.
- Residual risk **[inference]**: a device whose wall clock is far in the future writes stamps that win over everything until real time catches up. Mitigation options for the sync-protocol ticket: cap acceptable drift on apply and flag/clamp outliers.

## 4. Deletion and tombstones

- Remove-as-add-to-a-second-set: Wuu and Bernstein's set keeps an add set A and a removal set R ("R is often called the tombstone set") with value A \ R; their construct (the U-Set) "assume[s] that every element is unique and added only once." ([Shapiro et al., RR-7687](https://inria.hal.science/inria-00609399)) With client-generated UUIDs, element uniqueness holds for tasks; re-creating a task is a new UUID.
- Concurrent add/remove of the same element admits alternative semantics (add wins, remove wins, last update wins) ([Shapiro et al.](https://inria.hal.science/inria-00609399)). The policy for **edit vs concurrent delete** is a product decision. Syncthing, for reference, keeps the modified file as a conflict copy when a deletion wins ([Syncthing docs](https://docs.syncthing.net/users/syncing.html)). A task-app analogue is "edit after delete resurrects"; for the data-model/sync-protocol tickets to decide.
- Garbage collection is the hard part: Wuu and Bernstein's log GC works only with knowledge of what every peer has seen ([Shapiro et al.](https://inria.hal.science/inria-00609399)); a folder has no membership list, so drop tombstones only after all known devices have confirmed past them, or keep them forever (cheap for a personal task list) **[inference]**.
- Schema implication: use soft delete (`deleted_at` / tombstone stamp) on synced tables instead of hard `DELETE ... ON DELETE CASCADE`. Cascades must be re-expressed as explicit tombstones for children **[inference]**. cr-sqlite handles row existence with causal length rather than tombstone rows ([README](https://github.com/vlcn-io/cr-sqlite)).

## 5. What sync requires of row identity

- Auto-increment ids are allocated locally; two offline devices both mint `id = N` for different tasks **[inference]**. Every Zenith table uses `INTEGER PRIMARY KEY AUTOINCREMENT` and has FKs to those integers (`internal/storage/schema.sql`).
- Both mature mechanisms key on primary key: the session extension identifies rows by PRIMARY KEY ([session docs](https://www.sqlite.org/sessionintro.html)) and cr-sqlite requires a non-nullable PK that is sent over the wire as the row identity ([README](https://github.com/vlcn-io/cr-sqlite); [tableinfo.rs](https://github.com/vlcn-io/cr-sqlite/blob/main/core/rs/core/src/tableinfo.rs)).
- UUID choice: RFC 9562 UUIDv7 is a 48-bit Unix-ms timestamp plus random bits, and "time-ordered monotonic UUIDs benefit from greater database-index locality," while non-time-ordered versions (v4) have "poor database-index locality." It also recommends time-based UUIDs as surrogate keys, not name-based ones as natural keys. ([RFC 9562](https://www.rfc-editor.org/rfc/rfc9562.html), sections 2.1, 5.7, 6.11, 6.13)
- Migration mechanics (Zenith-specific, per `migrate.go` convention):
  - `ALTER TABLE ... ADD COLUMN` cannot add a column with `PRIMARY KEY` or `UNIQUE`, and `NOT NULL` needs a non-NULL default. ([SQLite ALTER TABLE](https://www.sqlite.org/lang_altertable.html)) So add a nullable `uuid` column, backfill, then enforce uniqueness with `CREATE UNIQUE INDEX`. Changing the actual primary key (or rewriting FKs from int to uuid) needs SQLite's table-rebuild procedure (create new table, copy, drop, rename; foreign keys off, `PRAGMA foreign_key_check`). ([SQLite ALTER TABLE](https://www.sqlite.org/lang_altertable.html))
  - Two-ID approach is the lighter option **[inference]**: keep integer `id` as a device-local handle (the CLI's `zenith task done 5` UX depends on it) and add `uuid` as the sync identity, with join tables (`task_tags`) and FKs resolved through uuid in the op-log.
- Pitfalls specific to this codebase:
  - `ensureDefaultProject` inserts an "Anytime" project when `projects` is empty (`internal/storage/sqlite_store.go`, `CLAUDE.md`). Two devices that each run it first would create two "Anytime" projects with different uuids after sync. The default project needs a fixed, well-known identity.
  - `tags.name` is UNIQUE; two devices creating "work" offline yield two tag uuids with the same name. Needs a merge rule (e.g. deterministic uuid derived from name, which RFC 9562 allows via name-based UUIDs, or merge-by-name on apply).
  - Device-local state should not sync: `scheduled_timers` (pending notifications fire per device by `checkExpiredTimersPassive` and the `daemon`) and probably running `task_time_logs` (`end_time IS NULL`). Which tables sync is a data-model decision **[inference]**.
  - Append-only tables (`task_history`, `habit_logs`, finished `task_time_logs`) map naturally onto a grow-only set with uuid keys and need no per-field LWW **[inference]**.

## 6. Pitfalls by folder provider

| Provider | Documented behavior relevant to us | Source |
|---|---|---|
| Dropbox | Conflicted copies on simultaneous / offline / left-open-file edits; copy is named with editor + date; last-saved version becomes the conflicted copy; advises moving a file out of Dropbox while editing to avoid conflicts. | [Dropbox help](https://help.dropbox.com/organize/conflicted-copy) |
| iCloud Drive | Conflict versions chosen by most-recent modification; same-name new files in Documents get numeric suffix ("bounced"); stale versions sync to peers and count against quota; apps should read/write through `NSFileCoordinator`. "Optimize Mac Storage" can move documents to iCloud when space is needed (files then show as cloud-only). | [TN2336](https://developer.apple.com/library/archive/technotes/tn2336/_index.html); [File System Programming Guide: iCloud](https://developer.apple.com/library/archive/documentation/FileManagement/Conceptual/FileSystemProgrammingGuide/iCloud/iCloud.html); [Apple Support: iCloud Drive](https://support.apple.com/guide/mac-help/store-files-in-icloud-drive-mchle5a61431/mac) |
| Google Drive | Desktop client supports Windows and macOS, **not Linux**. "Stream" mode keeps files primarily in the cloud and offline only when accessed; "mirror" always keeps files locally; Google notes "some applications use a combination of APIs that make files difficult to stream." On mismatch when switching modes, "Drive for desktop keeps both." | [Drive system requirements](https://support.google.com/drive/answer/2375082); [Stream & mirror](https://support.google.com/drive/answer/13401938) |
| Syncthing | Conflict rule by mtime then device id; conflict files propagate; temp files `.syncthing.*.tmp` / `~syncthing~*.tmp` are hidden and ignored; `.stignore` supports ignore patterns; works on all three desktop OSes (not separately verified here). | [Syncing](https://docs.syncthing.net/users/syncing.html); [Ignoring](https://docs.syncthing.net/users/ignoring.html) |

Implications for the design **[inference]**:
- Never open a file in the sync folder with a long-lived handle; write temp-then-rename under a name pattern that is not Syncthing's temp prefix.
- Expect cloud-only/evicted files (iCloud, Drive streaming): reading a not-yet-downloaded file may block or fail, so the sync step must be asynchronous, time-bounded and retryable, never in the CLI's hot path. The sources I read do not describe the exact placeholder/download behavior on read (the Apple guide fetched does not cover it), so exact behavior is **[unverified]**; recommend the sync ticket test on real machines. The user should be told to enable "Keep Downloaded" / mirror mode. Apple Support's page I fetched did not state the "Keep Downloaded" option; it surfaced in a search summary only, **[unverified]**.
- Expect and ignore conflict-copy files (`*conflict*`, `*conflicted copy*`, ` 2.seg`), treating any well-formed segment as valid input (idempotent apply makes duplicates harmless).
- Offer a folder-health check on first setup (writable? locally materialized?), because Google Drive has no Linux client and each provider is a different product.
- Do not assume any provider preserves modification times or ordering across devices; order by HLC inside the data, never by file mtime.

## 7. What to carry into the data-model decision (#6)

I'm not making that decision. These are the facts and constraints the model should account for.

1. **Identity**: every synced entity needs a client-generated stable UUID (v7 is the RFC-recommended time-ordered form). Decide the relationship between local integer `id` (CLI UX) and the sync `uuid`, and the plan for retrofitting existing user DBs via `migrate.go` (ALTER ADD COLUMN limits above). Give the default "Anytime" project a fixed identity.
2. **Mutability model**: choose which fields are individually mergeable (per-field LWW) and which entities are append-only (history, logs). Record per-field version stamps `(hlc, device_id)`; `updated_at CURRENT_TIMESTAMP` cannot do this job.
3. **Deletion**: soft delete / tombstones on synced tables; no reliance on `ON DELETE CASCADE` to propagate deletes; decide edit-vs-delete policy and tombstone retention.
4. **Synced vs device-local**: classify tables (`scheduled_timers`, running timers, `tasks.recurring` side effects). Don't let device-local notification state sync.
5. **Constraints that don't merge**: `UNIQUE` columns (`tags.name`) and FK enforcement need an explicit merge/repair rule. If cr-sqlite stays an option, these must be removed or relaxed up front.
6. **Keep the engine choice open if cheap**: if the model has uuid PKs, no non-PK unique indexes, and defaults on NOT NULL columns, it would also satisfy cr-sqlite's checks, preserving the option to adopt it later **[inference]**. Conversely, committing to cr-sqlite now binds three front-ends and three OS builds to a native extension whose last tagged release is January 2024.
7. **Store API boundary**: writes should go through one change-capturing layer (the `Store` interface) so CLI, TUI and desktop all emit ops; direct SQL elsewhere would bypass sync **[inference]**.
8. **DB placement**: the live DB stays at `~/.zenith/data.db` (local), with the sync folder holding only op segments. This is a prerequisite assumption for everything above.

## 8. Open questions not resolved by primary sources

- Whether Dropbox, iCloud Drive and Google Drive deliver a freshly written file atomically to peers (readers could see partial content). Only Syncthing documents its temp-file replacement. Test empirically.
- Exact on-read behavior for evicted/cloud-only files on iCloud and Drive-stream on each OS.
- Linux options for iCloud (no first-party client found in my sources) and Dropbox/Syncthing parity on Linux and Windows.
- Whether cr-sqlite development is effectively paused (last tag 2024-01, last push 2026-08); the repo exposes no explicit status statement.
- HLC behavior under adversarial or badly wrong clocks (the paper's bounds assume NTP-class synchronization).
