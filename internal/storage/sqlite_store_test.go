package storage

import (
	"database/sql"
	"os"
	"testing"
	"time"

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

func TestSQLiteStore_ScheduledTimers(t *testing.T) {
	tempFile := "test_timers.db"
	defer os.Remove(tempFile)

	db, err := NewSQLiteStoreAtPath(tempFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer db.Close()

	// Add scheduled timer
	now := time.Now()
	timer := &models.ScheduledTimer{
		Title:       "Test Timer Reminder",
		TriggerTime: now.Add(1 * time.Minute),
	}

	if err := db.AddScheduledTimer(timer); err != nil {
		t.Fatalf("failed to add scheduled timer: %v", err)
	}

	if timer.ID == 0 {
		t.Errorf("expected non-zero ID for added timer")
	}

	// Retrieve pending timers
	timers, err := db.GetPendingScheduledTimers()
	if err != nil {
		t.Fatalf("failed to get pending timers: %v", err)
	}

	if len(timers) != 1 || timers[0].Title != "Test Timer Reminder" {
		t.Errorf("expected 1 pending timer with matching title, got %d", len(timers))
	}

	// Mark triggered
	if err := db.MarkScheduledTimerTriggered(timer.ID); err != nil {
		t.Fatalf("failed to mark timer triggered: %v", err)
	}

	// Verify no pending timers remain
	timers, err = db.GetPendingScheduledTimers()
	if err != nil {
		t.Fatalf("failed to get pending timers after trigger: %v", err)
	}

	if len(timers) != 0 {
		t.Errorf("expected 0 pending timers, got %d", len(timers))
	}
}

func TestSQLiteStore_TaskHistory(t *testing.T) {
	tempFile := "test_history.db"
	defer os.Remove(tempFile)

	db, err := NewSQLiteStoreAtPath(tempFile)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}
	defer db.Close()

	// 1. Add a task and verify it creates an 'created' history log
	task := &models.Task{
		Title:  "History Test Task",
		Status: "todo",
	}
	if err := db.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	history, err := db.GetTaskHistory(task.ID)
	if err != nil {
		t.Fatalf("failed to get task history: %v", err)
	}

	if len(history) != 1 || history[0].Action != "created" {
		t.Errorf("expected 1 history record with action 'created', got %d", len(history))
	}

	// 2. Start/Stop timer and verify history entries are added
	if err := db.StartTaskTimer(task.ID); err != nil {
		t.Fatalf("failed to start task timer: %v", err)
	}
	if err := db.StopTaskTimer(task.ID); err != nil {
		t.Fatalf("failed to stop task timer: %v", err)
	}

	history, err = db.GetTaskHistory(task.ID)
	if err != nil {
		t.Fatalf("failed to get task history: %v", err)
	}

	// We expect: created, started_timer, stopped_timer (sorted DESC by timestamp)
	if len(history) != 3 {
		t.Errorf("expected 3 history records, got %d", len(history))
	}

	// 3. Update task status and verify status_changed log
	task.Status = "done"
	if err := db.UpdateTask(task); err != nil {
		t.Fatalf("failed to update task: %v", err)
	}

	history, err = db.GetTaskHistory(task.ID)
	if err != nil {
		t.Fatalf("failed to get task history: %v", err)
	}

	if len(history) != 4 || history[0].Action != "status_changed" {
		t.Errorf("expected 4 history records, first one 'status_changed', got %d", len(history))
	}
}


