package storage

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/orderkey"
)

func migrate(db *sql.DB) error {
	// 1. Add project_id to tasks if it doesn't exist
	if !columnExists(db, "tasks", "project_id") {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN project_id INTEGER REFERENCES projects(id) ON DELETE SET NULL")
		if err != nil {
			return fmt.Errorf("failed to add project_id column: %w", err)
		}
	}

	// 2. Add priority to tasks if it doesn't exist
	if !columnExists(db, "tasks", "priority") {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN priority INTEGER DEFAULT 0")
		if err != nil {
			return fmt.Errorf("failed to add priority column: %w", err)
		}
	}

	// 3. Add due_date to tasks if it doesn't exist
	// (Skipped once When/Deadline exist: the columns were migrated away and must not come back.)
	if !columnExists(db, "tasks", "due_date") && !columnExists(db, "tasks", "when_kind") {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN due_date DATETIME")
		if err != nil {
			return fmt.Errorf("failed to add due_date column: %w", err)
		}
	}

	// 4. Add planned_date to tasks if it doesn't exist
	if !columnExists(db, "tasks", "planned_date") && !columnExists(db, "tasks", "when_kind") {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN planned_date DATE")
		if err != nil {
			return fmt.Errorf("failed to add planned_date column: %w", err)
		}
	}

	// 5. Add recurring to tasks if it doesn't exist
	if !columnExists(db, "tasks", "recurring") && !columnExists(db, "tasks", "repeater_id") {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN recurring TEXT DEFAULT 'none'")
		if err != nil {
			return fmt.Errorf("failed to add recurring column: %w", err)
		}
	}

	// 6. Create scheduled_timers table if it doesn't exist
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS scheduled_timers (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER,
		title TEXT NOT NULL,
		trigger_time DATETIME NOT NULL,
		status TEXT DEFAULT 'pending',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
	)`)
	if err != nil {
		return fmt.Errorf("failed to create scheduled_timers table: %w", err)
	}
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_scheduled_timers_status ON scheduled_timers(status)")

	// 7. Create task_history table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS task_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		task_id INTEGER NOT NULL,
		action TEXT NOT NULL,
		details TEXT NOT NULL,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
	)`)
	if err != nil {
		return fmt.Errorf("failed to create task_history table: %w", err)
	}
	_, _ = db.Exec("CREATE INDEX IF NOT EXISTS idx_task_history_task ON task_history(task_id)")

	// 8. Sync identity: a uuid on every entity table (ADR 0003).
	if err := migrateUUIDs(db); err != nil {
		return err
	}

	// 9. Status model and Subtasks (ADR 0005).
	if err := migrateStatusAndSubtasks(db); err != nil {
		return err
	}

	// 10. Area > Project > Heading > Task, and the retired Anytime project.
	if err := migrateHierarchy(db); err != nil {
		return err
	}

	// 11. Tags are unique by normalized name in the core, not by constraint.
	if err := migrateTagsDropUnique(db); err != nil {
		return err
	}

	// 12. When / Deadline replace planned_date / due_date (ADR 0004).
	if err := migrateWhenDeadline(db); err != nil {
		return err
	}

	// 13. Manual ordering by fractional index (ADR 0007).
	if err := migrateSortKeys(db); err != nil {
		return err
	}

	// 14. Repeaters replace the single-row recurring column (ADR 0006).
	if err := migrateRepeaters(db); err != nil {
		return err
	}

	// 15. Sync groundwork: tombstone columns and initial field stamps (ADR 0010).
	return migrateSyncGroundwork(db)
}

// uuidTables are the tables with an integer `id` handle. The task_tags join
// table has no id of its own; sync identifies its rows by (task uuid, tag uuid).
var uuidTables = []string{
	"areas", "headings", "repeaters", "projects", "tasks", "tags", "task_time_logs",
	"habits", "habit_logs", "scheduled_timers", "task_history",
}

// uuidSQL generates a random version 4 UUID inside SQLite so a whole table can
// be backfilled with one statement (atomic, and safe if two processes migrate
// at once). randomblob is evaluated per row.
const uuidSQL = `lower(hex(randomblob(4)) || '-' || hex(randomblob(2)) || '-4' || substr(hex(randomblob(2)), 2) || '-' || substr('89ab', 1 + (abs(random()) % 4), 1) || substr(hex(randomblob(2)), 2) || '-' || hex(randomblob(6)))`

// migrateUUIDs is idempotent: it adds the column only when missing, backfills
// only rows without a uuid, and keeps existing uuids untouched.
func migrateUUIDs(db *sql.DB) error {
	for _, table := range uuidTables {
		if !columnExists(db, table, "id") {
			continue // table not created yet (partial legacy database)
		}
		if !columnExists(db, table, "uuid") {
			_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN uuid TEXT", table))
			// A concurrent process may have added it between the check and the ALTER.
			if err != nil && !columnExists(db, table, "uuid") {
				return fmt.Errorf("failed to add uuid column to %s: %w", table, err)
			}
		}
		if _, err := db.Exec(fmt.Sprintf("UPDATE %s SET uuid = %s WHERE uuid IS NULL OR uuid = ''", table, uuidSQL)); err != nil {
			return fmt.Errorf("failed to backfill uuids in %s: %w", table, err)
		}
		if _, err := db.Exec(fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS idx_%s_uuid ON %s(uuid)", table, table)); err != nil {
			return fmt.Errorf("failed to index uuids in %s: %w", table, err)
		}
	}
	return nil
}

func columnExists(db *sql.DB, tableName, columnName string) bool {
	query := fmt.Sprintf("PRAGMA table_info(%s)", tableName)
	rows, err := db.Query(query)
	if err != nil {
		return false
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var dtype string
		var notnull int
		var dfltValue interface{}
		var pk int
		if err := rows.Scan(&cid, &name, &dtype, &notnull, &dfltValue, &pk); err != nil {
			continue
		}
		if name == columnName {
			return true
		}
	}
	return false
}

// migrateStatusAndSubtasks moves tasks to Status = open | completed | cancelled
// and adds the parent container column. Both steps are idempotent.
//
// Legacy 'done' becomes completed; everything else that is not already a new
// value (todo, in_progress, ...) becomes open. A task that was in progress keeps
// its running time log, which is what now makes it "in progress".
func migrateStatusAndSubtasks(db *sql.DB) error {
	if !columnExists(db, "tasks", "id") {
		return nil
	}
	if !columnExists(db, "tasks", "parent_task_id") {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN parent_task_id INTEGER REFERENCES tasks(id) ON DELETE CASCADE")
		if err != nil && !columnExists(db, "tasks", "parent_task_id") {
			return fmt.Errorf("failed to add parent_task_id column: %w", err)
		}
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_parent ON tasks(parent_task_id)"); err != nil {
		return fmt.Errorf("failed to index parent_task_id: %w", err)
	}
	if !columnExists(db, "tasks", "status") {
		return nil // partial legacy table; the schema step adds it with the default
	}
	_, err := db.Exec(`UPDATE tasks SET status = CASE
			WHEN lower(status) IN ('done', 'completed') THEN 'completed'
			WHEN lower(status) IN ('cancelled', 'canceled') THEN 'cancelled'
			ELSE 'open' END
		WHERE status IS NULL OR status NOT IN ('open', 'completed', 'cancelled')`)
	if err != nil {
		return fmt.Errorf("failed to migrate task statuses: %w", err)
	}
	return nil
}

// addColumn adds a column once; a concurrent process may win the race.
func addColumn(db *sql.DB, table, column, ddl string) error {
	if columnExists(db, table, column) {
		return nil
	}
	_, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, ddl))
	if err != nil && !columnExists(db, table, column) {
		return fmt.Errorf("failed to add %s.%s: %w", table, column, err)
	}
	return nil
}

// migrateHierarchy adds the container columns for Area > Project > Heading >
// Task and retires the seeded 'Anytime' project: its tasks become project-less
// and the row is removed. Only the exact seed (name and description) is
// touched, so a user's own project is never deleted. Idempotent.
func migrateHierarchy(db *sql.DB) error {
	if !columnExists(db, "tasks", "id") || !columnExists(db, "projects", "id") {
		return nil // partial legacy database
	}
	for _, c := range []struct{ table, column, ddl string }{
		{"projects", "area_id", "INTEGER REFERENCES areas(id) ON DELETE SET NULL"},
		{"projects", "status", "TEXT DEFAULT 'open'"},
		{"tasks", "area_id", "INTEGER REFERENCES areas(id) ON DELETE SET NULL"},
		{"tasks", "heading_id", "INTEGER REFERENCES headings(id) ON DELETE CASCADE"},
	} {
		if err := addColumn(db, c.table, c.column, c.ddl); err != nil {
			return err
		}
	}
	for _, stmt := range []string{
		"CREATE INDEX IF NOT EXISTS idx_projects_area ON projects(area_id)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_area ON tasks(area_id)",
		"CREATE INDEX IF NOT EXISTS idx_tasks_heading ON tasks(heading_id)",
		"CREATE INDEX IF NOT EXISTS idx_headings_project ON headings(project_id)",
		"UPDATE projects SET status = 'open' WHERE status IS NULL OR status = ''",
		`UPDATE tasks SET project_id = NULL WHERE project_id IN
			(SELECT id FROM projects WHERE name = 'Anytime' AND description = 'Default project for tasks')`,
		"DELETE FROM projects WHERE name = 'Anytime' AND description = 'Default project for tasks'",
	} {
		if _, err := db.Exec(stmt); err != nil {
			return fmt.Errorf("hierarchy migration failed on %q: %w", stmt, err)
		}
	}
	return nil
}

// migrateTagsDropUnique removes the UNIQUE constraint on tags.name (ADR 0010).
// SQLite cannot drop a constraint in place, so the table is rebuilt with every
// id, uuid and attachment preserved. It does nothing once the constraint is gone.
func migrateTagsDropUnique(db *sql.DB) error {
	if !columnExists(db, "tags", "uuid") {
		return nil
	}
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master
		WHERE type = 'index' AND tbl_name = 'tags' AND name LIKE 'sqlite_autoindex_tags_%'`).Scan(&n)
	if err != nil || n == 0 {
		return err
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, stmt := range []string{
		`CREATE TABLE tags_rebuilt (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			color TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			uuid TEXT
		)`,
		"INSERT INTO tags_rebuilt (id, name, color, created_at, uuid) SELECT id, name, color, created_at, uuid FROM tags",
		"DROP TABLE tags",
		"ALTER TABLE tags_rebuilt RENAME TO tags",
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_tags_uuid ON tags(uuid)",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return fmt.Errorf("failed to rebuild tags without UNIQUE name: %w", err)
		}
	}
	return tx.Commit()
}

var datePrefix = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}`)

// legacyDate reduces whatever the old date columns held (a time.Time, or text
// in several shapes written by earlier versions) to a plain YYYY-MM-DD, or "".
func legacyDate(v any) string {
	switch d := v.(type) {
	case time.Time:
		return d.Format(models.DateLayout)
	case string:
		return datePrefix.FindString(d)
	case []byte:
		return datePrefix.FindString(string(d))
	}
	return ""
}

// migrateWhenDeadline maps the free-form legacy dates onto When and Deadline:
// planned_date becomes scheduled(date) and due_date becomes the Deadline. The
// old columns are then dropped. Idempotent: it does nothing once they are gone.
func migrateWhenDeadline(db *sql.DB) error {
	if !columnExists(db, "tasks", "id") {
		return nil
	}
	for _, c := range []struct{ column, ddl string }{
		{"when_kind", "TEXT DEFAULT 'anytime'"},
		{"when_date", "TEXT"},
		{"deadline", "TEXT"},
		{"evening", "INTEGER DEFAULT 0"},
	} {
		if err := addColumn(db, "tasks", c.column, c.ddl); err != nil {
			return err
		}
	}
	hasPlanned, hasDue := columnExists(db, "tasks", "planned_date"), columnExists(db, "tasks", "due_date")
	if !hasPlanned && !hasDue {
		return nil
	}
	plannedExpr, dueExpr := "NULL", "NULL"
	if hasPlanned {
		plannedExpr = "planned_date"
	}
	if hasDue {
		dueExpr = "due_date"
	}
	rows, err := db.Query("SELECT id, " + plannedExpr + ", " + dueExpr + " FROM tasks")
	if err != nil {
		return err
	}
	type mapped struct {
		id           int64
		planned, due string
	}
	var all []mapped
	for rows.Next() {
		var id int64
		var planned, due any
		if err := rows.Scan(&id, &planned, &due); err != nil {
			rows.Close()
			return err
		}
		all = append(all, mapped{id, legacyDate(planned), legacyDate(due)})
	}
	rows.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, m := range all {
		kind, date := "anytime", any(nil)
		if m.planned != "" {
			kind, date = "scheduled", m.planned
		}
		if _, err := tx.Exec("UPDATE tasks SET when_kind = ?, when_date = ?, deadline = ?, evening = 0 WHERE id = ?",
			kind, date, nullIfEmpty(m.due), m.id); err != nil {
			return err
		}
	}
	for _, stmt := range []string{"DROP INDEX IF EXISTS idx_tasks_planned_date"} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}
	// The pool has one connection, so existence was decided before the transaction began.
	for col, present := range map[string]bool{"planned_date": hasPlanned, "due_date": hasDue} {
		if present {
			if _, err := tx.Exec("ALTER TABLE tasks DROP COLUMN " + col); err != nil {
				return fmt.Errorf("failed to drop tasks.%s: %w", col, err)
			}
		}
	}
	return tx.Commit()
}

// migrateSortKeys adds sort_key to the orderable tables and numbers rows that
// have none in their existing (id) order, so today's apparent order is kept.
// Rows that already have a key are left alone; new numbering continues after them.
func migrateSortKeys(db *sql.DB) error {
	for table := range sortKeyTables {
		if !columnExists(db, table, "id") {
			continue
		}
		if err := addColumn(db, table, "sort_key", "TEXT"); err != nil {
			return err
		}
		rows, err := db.Query("SELECT id FROM " + table + " WHERE sort_key IS NULL OR sort_key = '' ORDER BY id")
		if err != nil {
			return err
		}
		var ids []int64
		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err != nil {
				rows.Close()
				return err
			}
			ids = append(ids, id)
		}
		rows.Close()
		if len(ids) == 0 {
			continue
		}
		var max sql.NullString
		if err := db.QueryRow("SELECT MAX(sort_key) FROM " + table).Scan(&max); err != nil {
			return err
		}
		keys := orderkey.Initial(len(ids))
		if max.String != "" { // continue after what exists
			prev := max.String
			for i := range keys {
				k, err := orderkey.Between(prev, "")
				if err != nil {
					return err
				}
				keys[i], prev = k, k
			}
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		for i, id := range ids {
			if _, err := tx.Exec("UPDATE "+table+" SET sort_key = ? WHERE id = ?", keys[i], id); err != nil {
				tx.Rollback()
				return err
			}
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// migrateRepeaters turns every legacy daily/weekly/monthly task into a Repeater
// with the equivalent rule, links the task as its current occurrence, and drops
// the recurring column. The anchor day is the task's scheduled day, else its
// deadline, else the day it was created. Idempotent: it does nothing once the
// column is gone.
func migrateRepeaters(db *sql.DB) error {
	if !columnExists(db, "tasks", "id") {
		return nil
	}
	for _, c := range []struct{ column, ddl string }{{"repeater_id", "INTEGER"}, {"occurrence_date", "TEXT"}} {
		if err := addColumn(db, "tasks", c.column, c.ddl); err != nil {
			return err
		}
	}
	if _, err := db.Exec("CREATE INDEX IF NOT EXISTS idx_tasks_repeater ON tasks(repeater_id, occurrence_date)"); err != nil {
		return err
	}
	if !columnExists(db, "tasks", "recurring") || !columnExists(db, "tasks", "created_at") {
		return nil // nothing to convert (or a partial table with no data to read)
	}
	rows, err := db.Query(`SELECT id, recurring, when_date, deadline, created_at, title, description, project_id, area_id, heading_id, priority, evening
		FROM tasks WHERE recurring IN ('daily', 'weekly', 'monthly')`)
	if err != nil {
		return err
	}
	type legacy struct {
		id                  int64
		recurring           string
		whenDate, deadline  sql.NullString
		created             any
		title, desc         sql.NullString
		project, area, head sql.NullInt64
		priority            int
		evening             sql.NullBool
	}
	var all []legacy
	for rows.Next() {
		var l legacy
		if err := rows.Scan(&l.id, &l.recurring, &l.whenDate, &l.deadline, &l.created, &l.title, &l.desc, &l.project, &l.area, &l.head, &l.priority, &l.evening); err != nil {
			rows.Close()
			return err
		}
		all = append(all, l)
	}
	rows.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	for _, l := range all {
		anchor := l.whenDate.String
		if anchor == "" {
			anchor = l.deadline.String
		}
		if anchor == "" {
			anchor = legacyDate(l.created)
		}
		day, err := time.Parse(models.DateLayout, anchor)
		if err != nil {
			day = time.Now()
			anchor = day.Format(models.DateLayout)
		}
		kind, weekdays, monthDay := "days", "", 0
		switch l.recurring {
		case "weekly":
			kind, weekdays = "weeks", strconv.Itoa(int(day.Weekday()))
		case "monthly":
			kind, monthDay = "months", day.Day()
		}
		res, err := tx.Exec(`INSERT INTO repeaters (uuid, kind, interval, weekdays, month_day, title, description, project_id, area_id, heading_id, priority, evening)
			VALUES (?, ?, 1, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			newUUID(), kind, weekdays, monthDay, l.title.String, l.desc.String, nullInt(l.project), nullInt(l.area), nullInt(l.head), l.priority, l.evening.Bool)
		if err != nil {
			return err
		}
		rid, _ := res.LastInsertId()
		if _, err := tx.Exec("UPDATE tasks SET repeater_id = ?, occurrence_date = ? WHERE id = ?", rid, anchor, l.id); err != nil {
			return err
		}
	}
	if _, err := tx.Exec("ALTER TABLE tasks DROP COLUMN recurring"); err != nil {
		return fmt.Errorf("failed to drop tasks.recurring: %w", err)
	}
	return tx.Commit()
}

func nullInt(v sql.NullInt64) any {
	if !v.Valid {
		return nil
	}
	return v.Int64
}

// tombstoneTables are the synced tables with a stamped deleted_at (soft delete).
var tombstoneTables = []string{"areas", "projects", "headings", "tasks", "tags", "repeaters", "habits", "task_tags"}

// stampedEntities lists, for each table, the entity name and the fields that
// get an initial stamp. Grow-only rows (logs, history) carry a single "_created".
var stampedEntities = []struct {
	table, entity string
	fields        []string
}{
	{"areas", "area", areaCols},
	{"projects", "project", projectCols},
	{"headings", "heading", headingCols},
	{"tasks", "task", taskCols},
	{"tags", "tag", tagCols},
	{"repeaters", "repeater", repeaterCols},
	{"habits", "habit", habitCols},
	{"habit_logs", "habit_log", []string{"_created"}},
	{"task_history", "task_history", []string{"_created"}},
}

// migrateSyncGroundwork adds deleted_at to the synced tables and gives every
// existing row initial field stamps, so the first folder exchange has a version
// for everything. It only stamps rows that have none, so it is idempotent and
// cheap on every open.
func migrateSyncGroundwork(db *sql.DB) error {
	for _, table := range tombstoneTables {
		if !columnExists(db, table, "id") && table != "task_tags" {
			continue
		}
		if table == "task_tags" && !columnExists(db, table, "task_id") {
			continue
		}
		if err := addColumn(db, table, "deleted_at", "TEXT"); err != nil {
			return err
		}
	}
	if !columnExists(db, "tasks", "uuid") || !columnExists(db, "field_versions", "entity") {
		return nil // partial legacy database
	}
	st := &SQLiteStore{db: db, q: db, sync: &syncCtx{now: time.Now}}
	return st.WithTx(func(tx Store) error {
		return tx.(*SQLiteStore).stampMissing()
	})
}

// stampMissing stamps every row that has no stamps yet, including finished time
// logs and live tag memberships.
func (s *SQLiteStore) stampMissing() error {
	type pending struct {
		entity, uuid string
		fields       []string
	}
	var todo []pending
	collect := func(query string, entity string, fields []string, args ...any) error {
		rows, err := s.q.Query(query, args...)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var u sql.NullString
			if err := rows.Scan(&u); err != nil {
				return err
			}
			if u.Valid {
				todo = append(todo, pending{entity, u.String, fields})
			}
		}
		return rows.Err()
	}
	for _, e := range stampedEntities {
		if !columnExists2(s.q, e.table, "uuid") {
			continue
		}
		q := "SELECT uuid FROM " + e.table + " WHERE NOT EXISTS (SELECT 1 FROM field_versions f WHERE f.entity = ? AND f.uuid = " + e.table + ".uuid)"
		if err := collect(q, e.entity, e.fields, e.entity); err != nil {
			return err
		}
	}
	// Finished time logs are synced; a running one is device-local.
	if err := collect(`SELECT uuid FROM task_time_logs WHERE end_time IS NOT NULL AND NOT EXISTS
		(SELECT 1 FROM field_versions f WHERE f.entity = 'time_log' AND f.uuid = task_time_logs.uuid)`, "time_log", []string{"_created"}); err != nil {
		return err
	}
	if err := collect(`SELECT t.uuid || ':' || g.uuid FROM task_tags m JOIN tasks t ON t.id = m.task_id JOIN tags g ON g.id = m.tag_id
		WHERE m.deleted_at IS NULL AND NOT EXISTS
		(SELECT 1 FROM field_versions f WHERE f.entity = 'task_tag' AND f.uuid = t.uuid || ':' || g.uuid)`, "task_tag", []string{"deleted_at"}); err != nil {
		return err
	}
	for _, p := range todo {
		if err := s.stamp(p.entity, p.uuid, p.fields...); err != nil {
			return err
		}
	}
	return nil
}

// columnExists2 is columnExists for a queryer (usable inside a transaction).
func columnExists2(q queryer, table, column string) bool {
	var n int
	err := q.QueryRow("SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?", table, column).Scan(&n)
	return err == nil && n > 0
}
