package storage

import (
	"database/sql"
	"os"
	"testing"

	"github.com/yalcinumut/zenith-cli/internal/models"
)

func TestSQLiteStore_TasksWithTagsAndTimers(t *testing.T) {
	// Use a temporary database file
	tempFile := "test_zenith.db"
	defer os.Remove(tempFile)

	db, err := NewSQLiteStoreAtPath(tempFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer db.Close()

	// 1. Add a task
	task := &models.Task{
		Title: "Test Task",
	}
	if err := db.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// 2. Add a tag and attach it
	tag := &models.Tag{Name: "Urgent", Color: "#FF0000"}
	if err := db.AddTag(tag); err != nil {
		t.Fatalf("failed to add tag: %v", err)
	}
	if err := db.AttachTagToTask(task.ID, tag.ID); err != nil {
		t.Fatalf("failed to attach tag: %v", err)
	}

	// 3. Start/Stop timer
	if err := db.StartTaskTimer(task.ID); err != nil {
		t.Fatalf("failed to start timer: %v", err)
	}
	if err := db.StopTaskTimer(task.ID); err != nil {
		t.Fatalf("failed to stop timer: %v", err)
	}

	// 4. Verify results
	tasks, err := db.GetTasks()
	if err != nil {
		t.Fatalf("failed to get tasks: %v", err)
	}

	found := false
	for _, tk := range tasks {
		if tk.ID == task.ID {
			found = true
			if len(tk.Tags) != 1 || tk.Tags[0].Name != "Urgent" {
				t.Errorf("expected 1 tag 'Urgent', got %d tags", len(tk.Tags))
			}
			// Timer should have some duration (though very small in test)
			if tk.IsRunning {
				t.Errorf("expected task timer to be stopped")
			}
		}
	}

	if !found {
		t.Errorf("task not found in database")
	}
}

func TestMigrate(t *testing.T) {
	tempFile := "test_migrate.db"
	defer os.Remove(tempFile)

	db, err := sql.Open("sqlite3", tempFile)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// 1. Create a tasks table missing Phase 2 columns
	_, err = db.Exec(`CREATE TABLE tasks (id INTEGER PRIMARY KEY, title TEXT NOT NULL)`)
	if err != nil {
		t.Fatalf("failed to create partial tasks table: %v", err)
	}

	// 2. Run migrate
	if err := migrate(db); err != nil {
		t.Fatalf("migrate failed: %v", err)
	}

	// 3. Verify columns exist
	columns := []string{"project_id", "priority", "due_date", "planned_date", "recurring"}
	for _, col := range columns {
		if !columnExists(db, "tasks", col) {
			t.Errorf("expected column %s to exist after migration", col)
		}
	}
}

