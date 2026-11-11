package storage

import (
	"database/sql"
	"fmt"
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
	if !columnExists(db, "tasks", "due_date") {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN due_date DATETIME")
		if err != nil {
			return fmt.Errorf("failed to add due_date column: %w", err)
		}
	}

	// 4. Add planned_date to tasks if it doesn't exist
	if !columnExists(db, "tasks", "planned_date") {
		_, err := db.Exec("ALTER TABLE tasks ADD COLUMN planned_date DATE")
		if err != nil {
			return fmt.Errorf("failed to add planned_date column: %w", err)
		}
	}

	// 5. Add recurring to tasks if it doesn't exist
	if !columnExists(db, "tasks", "recurring") {
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
	return migrateHierarchy(db)
}

// uuidTables are the tables with an integer `id` handle. The task_tags join
// table has no id of its own; sync identifies its rows by (task uuid, tag uuid).
var uuidTables = []string{
	"areas", "headings", "projects", "tasks", "tags", "task_time_logs",
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
