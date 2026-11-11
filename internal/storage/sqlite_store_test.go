package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
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
	start := time.Date(2026, 1, 2, 9, 0, 0, 0, time.UTC)
	if err := db.OpenTimeLog(task.ID, start); err != nil {
		t.Fatalf("failed to open time log: %v", err)
	}
	secs, closed, err := db.CloseOpenTimeLog(task.ID, start.Add(90*time.Second))
	if err != nil || !closed || secs != 90 {
		t.Fatalf("CloseOpenTimeLog = %d, %v, %v; want 90, true, nil", secs, closed, err)
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

	db, err := openDB(tempFile)
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

	task := &models.Task{Title: "History Test Task", Status: "todo"}
	if err := db.AddTask(task); err != nil {
		t.Fatalf("failed to add task: %v", err)
	}

	// The store records exactly what it is told; rules live in core.
	history, err := db.GetTaskHistory(task.ID)
	if err != nil || len(history) != 0 {
		t.Fatalf("expected no history from a plain insert, got %d (err %v)", len(history), err)
	}

	stamp := time.Date(2026, 1, 2, 9, 30, 0, 0, time.UTC)
	for _, action := range []string{"created", "edited"} {
		if err := db.AddTaskHistory(&models.TaskHistory{TaskID: task.ID, Action: action, Details: "d", Timestamp: stamp}); err != nil {
			t.Fatalf("failed to add history: %v", err)
		}
	}

	history, err = db.GetTaskHistory(task.ID)
	if err != nil {
		t.Fatalf("failed to get task history: %v", err)
	}
	if len(history) != 2 || history[0].Action != "edited" {
		t.Fatalf("expected 2 records newest first, got %+v", history)
	}
	if !history[0].Timestamp.Equal(stamp) {
		t.Errorf("timestamp = %v, want %v", history[0].Timestamp, stamp)
	}
}

func TestSQLiteStore_WithTxRollsBackOnError(t *testing.T) {
	db, err := NewSQLiteStoreAtPath(filepath.Join(t.TempDir(), "tx.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	boom := fmt.Errorf("boom")
	err = db.WithTx(func(tx Store) error {
		if err := tx.AddTask(&models.Task{Title: "rolled back", Status: "todo"}); err != nil {
			return err
		}
		return boom
	})
	if err != boom {
		t.Fatalf("WithTx err = %v, want boom", err)
	}
	tasks, _ := db.GetTasks()
	if len(tasks) != 0 {
		t.Errorf("expected rollback, found %d tasks", len(tasks))
	}

	if _, err := db.GetTask(999); err != ErrNotFound {
		t.Errorf("GetTask(999) err = %v, want ErrNotFound", err)
	}
}

func TestSQLiteStore_ConcurrentWritersShareOneFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")

	// Two independent stores stand in for two processes (CLI and desktop app).
	a, err := NewSQLiteStoreAtPath(path)
	if err != nil {
		t.Fatalf("open store a: %v", err)
	}
	defer a.Close()
	b, err := NewSQLiteStoreAtPath(path)
	if err != nil {
		t.Fatalf("open store b: %v", err)
	}
	defer b.Close()

	const perWriter = 50
	errs := make(chan error, 2*perWriter)
	var wg sync.WaitGroup
	for _, s := range []*SQLiteStore{a, b} {
		wg.Add(1)
		go func(s *SQLiteStore) {
			defer wg.Done()
			for i := 0; i < perWriter; i++ {
				if err := s.AddTask(&models.Task{Title: fmt.Sprintf("task %d", i)}); err != nil {
					errs <- err
				}
			}
		}(s)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Errorf("concurrent write failed: %v", err)
	}

	tasks, err := a.GetTasks()
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(tasks) != 2*perWriter {
		t.Errorf("expected %d tasks, got %d", 2*perWriter, len(tasks))
	}
}

func TestSQLiteStore_PathsWithURISyntax(t *testing.T) {
	dirs := []string{"with space", "hash#dir", "pct%20dir"}
	if runtime.GOOS != "windows" { // '?' is not a legal Windows filename character
		dirs = append(dirs, "q?dir")
	}
	for _, dir := range dirs {
		p := filepath.Join(t.TempDir(), dir, "data.db")
		s, err := NewSQLiteStoreAtPath(p)
		if err != nil {
			t.Errorf("%q: %v", dir, err)
			continue
		}
		if err := s.AddTask(&models.Task{Title: "x"}); err != nil {
			t.Errorf("%q: %v", dir, err)
		}
		s.Close()
		s2, err := NewSQLiteStoreAtPath(p)
		if err != nil {
			t.Errorf("%q reopen: %v", dir, err)
			continue
		}
		if _, err := os.Stat(p); err != nil {
			t.Errorf("%q: db not at path: %v", dir, err)
		}
		if ts, _ := s2.GetTasks(); len(ts) != 1 {
			t.Errorf("%q: file not at expected path / %d tasks", dir, len(ts))
		}
		s2.Close()
	}
}
