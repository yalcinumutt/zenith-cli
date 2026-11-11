package core_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
	_ "modernc.org/sqlite"
)

// Legacy-migration harness. Later model tickets extend it: freeze a new
// fixture under testdata/, call legacyDB to materialise it as a database
// file, open it with openAt (which runs the real migrations) and assert
// through the core API. Do not invent a parallel harness.

const legacyV012 = "legacy_v0_1_2.sql"

// legacyDB builds a database file from a frozen fixture without running any
// Zenith migration, exactly as an old install would have left it.
func legacyDB(t *testing.T, fixture string) string {
	t.Helper()
	script, err := os.ReadFile(filepath.Join("testdata", fixture))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "data.db")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(string(script)); err != nil {
		t.Fatalf("fixture %s: %v", fixture, err)
	}
	return path
}

// openAt opens path through the real store (schema + migrations) and wraps it
// in a core.
func openAt(t *testing.T, path string) *core.Core {
	t.Helper()
	st, err := storage.NewSQLiteStoreAtPath(path)
	if err != nil {
		t.Fatalf("open/migrate: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return core.New(st, nil, nil)
}

// rawUUIDs returns table -> id -> uuid, read behind the core's back.
func rawUUIDs(t *testing.T, path string, tables []string) map[string]map[int64]string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	out := map[string]map[int64]string{}
	for _, table := range tables {
		rows, err := db.Query("SELECT id, uuid FROM " + table)
		if err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		m := map[int64]string{}
		for rows.Next() {
			var id int64
			var u sql.NullString
			if err := rows.Scan(&id, &u); err != nil {
				t.Fatal(err)
			}
			m[id] = u.String
		}
		rows.Close()
		out[table] = m
	}
	return out
}

var entityTables = []string{"projects", "tasks", "tags", "task_time_logs", "habits", "habit_logs", "scheduled_timers", "task_history"}

var uuidV4 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

func TestLegacyMigrationBackfillsEveryEntityTable(t *testing.T) {
	path := legacyDB(t, legacyV012)
	openAt(t, path)

	seen := map[string]bool{}
	for table, rows := range rawUUIDs(t, path, entityTables) {
		if len(rows) == 0 {
			t.Errorf("%s: fixture rows disappeared", table)
		}
		for id, u := range rows {
			if !uuidV4.MatchString(u) {
				t.Errorf("%s id %d: uuid %q is not a v4 uuid", table, id, u)
			}
			if seen[u] {
				t.Errorf("%s id %d: duplicate uuid %s", table, id, u)
			}
			seen[u] = true
		}
	}
}

func TestLegacyMigrationIsIdempotent(t *testing.T) {
	path := legacyDB(t, legacyV012)
	openAt(t, path)
	first := rawUUIDs(t, path, entityTables)

	openAt(t, path) // second open re-runs every migration
	openAt(t, path)
	second := rawUUIDs(t, path, entityTables)

	for table, rows := range first {
		for id, u := range rows {
			if second[table][id] != u {
				t.Errorf("%s id %d: uuid changed %s -> %s", table, id, u, second[table][id])
			}
		}
		if len(second[table]) != len(rows) {
			t.Errorf("%s: row count changed %d -> %d", table, len(rows), len(second[table]))
		}
	}
}

func TestLegacyRowsAndIntegerHandlesSurvive(t *testing.T) {
	c := openAt(t, legacyDB(t, legacyV012))

	tasks, err := c.Tasks.List()
	if err != nil {
		t.Fatal(err)
	}
	want := map[int64]string{1: "Write report", 2: "Fix header", 5: "Ship landing page", 7: "Call the bank"}
	if len(tasks) != len(want) {
		t.Fatalf("got %d tasks, want %d", len(tasks), len(want))
	}
	for _, task := range tasks {
		if want[task.ID] != task.Title {
			t.Errorf("task %d = %q, want %q", task.ID, task.Title, want[task.ID])
		}
		if task.UUID == "" {
			t.Errorf("task %d has no uuid", task.ID)
		}
	}

	got, err := c.Tasks.Get(5)
	if err != nil || len(got.Tags) != 2 || got.TotalTime != 90 || got.Recurring != "weekly" {
		t.Errorf("task 5 = %+v, err %v (want 2 tags, 90s, weekly)", got, err)
	}

	projects, _ := c.Projects.List()
	if len(projects) != 2 || projects[0].ID != 1 || projects[1].ID != 3 {
		t.Errorf("projects = %+v (default project must not be duplicated)", projects)
	}
	habits, _ := c.Habits.List()
	if len(habits) != 2 || habits[1].ID != 3 {
		t.Errorf("habits = %+v", habits)
	}
	if pending, _ := c.Reminders.Pending(); len(pending) != 1 || pending[0].ID != 2 {
		t.Errorf("pending reminders = %+v", pending)
	}

	// The CLI's integer handles keep working on migrated rows...
	if err := c.Tasks.Complete(5); err != nil {
		t.Fatal(err)
	}
	if err := c.Habits.Log(3); err != nil {
		t.Fatal(err)
	}
	// ...and new rows continue the sequence rather than reusing handles.
	added, err := c.Tasks.Add(core.AddTaskInput{Title: "new"})
	if err != nil || added.ID != 8 {
		t.Errorf("new task id = %d, err %v; want 8", added.ID, err)
	}
}

func TestNewRowsGetUUIDsThroughTheCore(t *testing.T) {
	path := legacyDB(t, legacyV012)
	c := openAt(t, path)

	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "fresh"})
	project, _ := c.Projects.Add("Fresh project")
	habit, _ := c.Habits.Add("Fresh habit")
	for name, u := range map[string]string{"task": task.UUID, "project": project.UUID, "habit": habit.UUID} {
		if !uuidV4.MatchString(u) {
			t.Errorf("%s uuid = %q", name, u)
		}
	}

	// Rows created as side effects (history, time logs, logs, reminders, tags).
	if err := c.Tasks.StartTimer(task.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.Tasks.StopTimer(task.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.Tasks.Tag(task.ID, "fresh-tag", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.Habits.Log(habit.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Reminders.Add("later", 1<<40, nil); err != nil {
		t.Fatal(err)
	}

	seen := map[string]bool{}
	for table, rows := range rawUUIDs(t, path, entityTables) {
		for id, u := range rows {
			if !uuidV4.MatchString(u) || seen[u] {
				t.Errorf("%s id %d: bad or duplicate uuid %q", table, id, u)
			}
			seen[u] = true
		}
	}

	// An existing tag keeps its uuid when it is attached again.
	before, _ := c.Tasks.Get(task.ID)
	_ = c.Tasks.Tag(task.ID, "fresh-tag", "#fff")
	after, _ := c.Tasks.Get(task.ID)
	if before.Tags[0].UUID != after.Tags[0].UUID {
		t.Errorf("tag uuid changed on re-tag: %s -> %s", before.Tags[0].UUID, after.Tags[0].UUID)
	}
}
