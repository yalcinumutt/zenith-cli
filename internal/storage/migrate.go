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
