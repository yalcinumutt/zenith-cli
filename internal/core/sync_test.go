package core_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// stampMap returns field -> hlc for one entity row, read behind the core.
func stampMap(t *testing.T, path, entity, uuid string) map[string]string {
	t.Helper()
	db := rawDB(t, path)
	defer db.Close()
	rows, err := db.Query("SELECT field, hlc FROM field_versions WHERE entity = ? AND uuid = ?", entity, uuid)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var f, h string
		_ = rows.Scan(&f, &h)
		out[f] = h
	}
	return out
}

func stampCount(t *testing.T, path string) int { return countRows(t, path, "field_versions") }

func syncCore(t *testing.T) (*core.Core, string, *fakeClock) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "s.db")
	st, err := storage.NewSQLiteStoreAtPath(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	clock := &fakeClock{now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	return core.New(st, clock, nil), path, clock
}

func TestCreateStampsEveryFieldInTheSameTransaction(t *testing.T) {
	c, path, _ := syncCore(t)
	task := add(t, c, core.AddTaskInput{Title: "stamped", Deadline: "2026-04-01"})
	got := stampMap(t, path, "task", task.UUID)
	for _, f := range []string{"title", "status", "when_kind", "deadline", "project_id", "sort_key", "priority"} {
		if got[f] == "" {
			t.Errorf("field %s has no stamp: %v", f, got)
		}
	}
	// One created entity per kind of thing.
	area, _ := c.Areas.Add("A")
	proj, _ := c.Projects.Add("P", nil)
	head, _ := c.Headings.Add(proj.ID, "H")
	hab, _ := c.Habits.Add("Read")
	_ = c.Tasks.Tag(task.ID, "tag", "")
	tags, _ := c.Tags.List()
	for entity, uuid := range map[string]string{"area": area.UUID, "project": proj.UUID, "heading": head.UUID, "habit": hab.UUID, "tag": tags[0].UUID} {
		if len(stampMap(t, path, entity, uuid)) == 0 {
			t.Errorf("%s was created without stamps", entity)
		}
	}
	if m := stampMap(t, path, "task_tag", task.UUID+":"+tags[0].UUID); m["deleted_at"] == "" {
		t.Errorf("tag membership not stamped: %v", m)
	}
}

func TestUpdatesStampOnlyChangedFields(t *testing.T) {
	c, path, clock := syncCore(t)
	task := add(t, c, core.AddTaskInput{Title: "t"})
	before := stampMap(t, path, "task", task.UUID)

	clock.advance(time.Minute)
	if err := c.Tasks.SetDeadline(task.ID, "2026-05-05"); err != nil {
		t.Fatal(err)
	}
	after := stampMap(t, path, "task", task.UUID)
	for f, h := range after {
		if f == "deadline" {
			if h <= before[f] {
				t.Errorf("deadline stamp did not advance: %s -> %s", before[f], h)
			}
			continue
		}
		if h != before[f] {
			t.Errorf("unchanged field %s was re-stamped: %s -> %s", f, before[f], h)
		}
	}

	// A no-op write stamps nothing at all.
	n := len(stampMap(t, path, "task", task.UUID))
	snapshot := stampMap(t, path, "task", task.UUID)
	_ = c.Tasks.SetDeadline(task.ID, "2026-05-05")
	for f, h := range stampMap(t, path, "task", task.UUID) {
		if snapshot[f] != h {
			t.Errorf("an unchanged value re-stamped %s", f)
		}
	}
	_ = n
	for before := stampMap(t, path, "task", task.UUID); ; {
		clock.advance(time.Minute)
		_ = c.Tasks.Complete(task.ID)
		after := stampMap(t, path, "task", task.UUID)
		if after["status"] <= before["status"] || after["title"] != before["title"] {
			t.Errorf("complete should stamp status only: %v -> %v", before, after)
		}
		break
	}
}

func TestAFailedMutationRecordsNoStamps(t *testing.T) {
	c, path, _ := syncCore(t)
	parent := add(t, c, core.AddTaskInput{Title: "parent"})
	add(t, c, core.AddTaskInput{Title: "child", ParentID: &parent.ID})
	n := stampCount(t, path)

	if err := c.Tasks.Complete(parent.ID); !errors.Is(err, core.ErrOpenSubtasks) {
		t.Fatalf("setup: %v", err)
	}
	if _, err := c.Tasks.Add(core.AddTaskInput{Title: "bad", HeadingID: ptr(999)}); err == nil {
		t.Fatal("expected failure")
	}
	if err := c.Tasks.Tag(parent.ID, " ", ""); err == nil {
		t.Fatal("expected failure")
	}
	if got := stampCount(t, path); got != n {
		t.Errorf("failed mutations left %d stamps", got-n)
	}

	// Store level: stamps written inside a transaction that then fails are rolled back.
	st, _ := storage.NewSQLiteStoreAtPath(filepath.Join(t.TempDir(), "x.db"))
	defer st.Close()
	boom := errors.New("boom")
	err := st.WithTx(func(tx storage.Store) error {
		if err := tx.AddTask(&models.Task{Title: "doomed", Status: models.StatusOpen}); err != nil {
			return err
		}
		return boom
	})
	if err != boom {
		t.Fatal(err)
	}
	db := rawDB(t, filepath.Join(filepath.Dir(path), "x.db"))
	_ = db.Close()
}

func TestDeletesAreTombstonesNotRemovals(t *testing.T) {
	c, path, _ := syncCore(t)
	parent := add(t, c, core.AddTaskInput{Title: "parent"})
	child := add(t, c, core.AddTaskInput{Title: "child", ParentID: &parent.ID})
	_ = c.Tasks.Tag(parent.ID, "x", "")
	_ = c.Tasks.StartTimer(child.ID)
	_ = c.Tasks.StopTimer(child.ID)
	_, _ = c.Reminders.Add("r", time.Hour, &child.ID)

	if err := c.Tasks.Delete(parent.ID); err != nil {
		t.Fatal(err)
	}
	for table, want := range map[string]int{"tasks": 2, "task_tags": 1, "tags": 1, "task_history": 8} {
		_ = want
		if got := countRows(t, path, table); got == 0 {
			t.Errorf("%s rows vanished", table)
		}
	}
	if live := countWhere(t, path, "tasks", "deleted_at IS NULL"); live != 0 {
		t.Errorf("live tasks = %d", live)
	}
	if dead := countWhere(t, path, "tasks", "deleted_at IS NOT NULL"); dead != 2 {
		t.Errorf("tombstoned tasks = %d, want parent and child", dead)
	}
	if got := countWhere(t, path, "task_tags", "deleted_at IS NOT NULL"); got != 1 {
		t.Errorf("tombstoned memberships = %d", got)
	}
	// Tombstones are stamped, finished time logs and history survive (grow-only).
	if m := stampMap(t, path, "task", parent.UUID); m["deleted_at"] == "" {
		t.Errorf("parent tombstone unstamped: %v", m)
	}
	if got := countRows(t, path, "task_time_logs"); got != 1 {
		t.Errorf("finished time log rows = %d, want 1", got)
	}
	if got := countRows(t, path, "scheduled_timers"); got != 0 {
		t.Errorf("reminders should be dropped with the task, have %d", got)
	}
	if _, err := c.Tasks.Get(child.ID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("deleted task still readable: %v", err)
	}
	// A tag can be attached again to a new task; the old membership stays dead.
	other := add(t, c, core.AddTaskInput{Title: "o"})
	if err := c.Tasks.Tag(other.ID, "X", ""); err != nil {
		t.Fatal(err)
	}
	if tags, _ := c.Tags.List(); len(tags) != 1 {
		t.Errorf("tags = %v", tags)
	}
}

func TestReattachingARemovedTagRevivesTheMembership(t *testing.T) {
	c, path, clock := syncCore(t)
	task := add(t, c, core.AddTaskInput{Title: "t"})
	_ = c.Tasks.Tag(task.ID, "home", "")
	tags, _ := c.Tags.List()
	key := task.UUID + ":" + tags[0].UUID
	first := stampMap(t, path, "task_tag", key)["deleted_at"]

	clock.advance(time.Minute)
	_ = c.Tasks.Untag(task.ID, "home")
	removed := stampMap(t, path, "task_tag", key)["deleted_at"]
	clock.advance(time.Minute)
	_ = c.Tasks.Tag(task.ID, "home", "")
	revived := stampMap(t, path, "task_tag", key)["deleted_at"]
	if !(first < removed && removed < revived) {
		t.Errorf("stamps not increasing: %s < %s < %s", first, removed, revived)
	}
	if got, _ := c.Tasks.Get(task.ID); len(got.Tags) != 1 {
		t.Errorf("tags = %+v", got.Tags)
	}
	if got := countRows(t, path, "task_tags"); got != 1 {
		t.Errorf("one membership row expected, have %d", got)
	}
}

func TestALiveTaskUnderATombstonedContainerIsContainerLess(t *testing.T) {
	c, path, _ := syncCore(t)
	proj, _ := c.Projects.Add("Doomed elsewhere", nil)
	area, _ := c.Areas.Add("Gone area")
	head, _ := c.Headings.Add(proj.ID, "H")
	inProject := add(t, c, core.AddTaskInput{Title: "in project", ProjectID: &proj.ID})
	inArea := add(t, c, core.AddTaskInput{Title: "in area", AreaID: &area.ID})
	underHeading := add(t, c, core.AddTaskInput{Title: "under heading", HeadingID: &head.ID})
	parent := add(t, c, core.AddTaskInput{Title: "p"})
	sub := add(t, c, core.AddTaskInput{Title: "sub", ParentID: &parent.ID})
	if got := list(t, c, core.ListInbox); len(got) != 1 {
		t.Fatalf("setup inbox = %v", got)
	}

	// Simulate a delete that arrived from another device: only the container is tombstoned.
	db := rawDB(t, path)
	for _, q := range []string{
		"UPDATE projects SET deleted_at = '2026-03-01 10:00:00'",
		"UPDATE areas SET deleted_at = '2026-03-01 10:00:00'",
		"UPDATE tasks SET deleted_at = '2026-03-01 10:00:00' WHERE title = 'p'",
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	for _, id := range []int64{inProject.ID, inArea.ID, underHeading.ID, sub.ID} {
		got, err := c.Tasks.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		if got.ProjectID != nil || got.AreaID != nil || got.HeadingID != nil || got.ParentID != nil {
			t.Errorf("task %d still reports a tombstoned container: %+v", id, got)
		}
	}
	inbox := list(t, c, core.ListInbox)
	sort.Strings(inbox)
	if len(inbox) != 4 || inbox[0] != "in area" || inbox[3] != "under heading" {
		t.Errorf("inbox = %v, want the four container-less tasks", inbox)
	}
	// The next edit repairs the stored reference (writes project = null, stamped).
	if err := c.Tasks.SetDeadline(inProject.ID, "2026-06-01"); err != nil {
		t.Fatal(err)
	}
	if m := stampMap(t, path, "task", inProject.UUID); m["project_id"] == "" || m["deadline"] == "" {
		t.Errorf("repair not stamped: %v", m)
	}
}

func TestDeviceLocalStateIsNeverStamped(t *testing.T) {
	c, path, _ := syncCore(t)
	task := add(t, c, core.AddTaskInput{Title: "t"})
	r, _ := c.Reminders.Add("ping", time.Minute, &task.ID)
	_ = c.Tasks.StartTimer(task.ID)

	db := rawDB(t, path)
	defer db.Close()
	count := func(where string) int {
		var n int
		_ = db.QueryRow("SELECT COUNT(*) FROM field_versions WHERE " + where).Scan(&n)
		return n
	}
	if count("entity IN ('scheduled_timer','reminder','sync_state','time_log')") != 0 {
		t.Error("reminders, the running time log or sync state were stamped")
	}
	_ = r
	_ = c.Tasks.StopTimer(task.ID) // becomes synced when stopped
	if count("entity = 'time_log'") != 1 {
		t.Error("a stopped time log should be stamped once")
	}
	var devices int
	_ = db.QueryRow("SELECT COUNT(*) FROM sync_state").Scan(&devices)
	if devices != 1 {
		t.Errorf("sync_state rows = %d", devices)
	}
}

func TestStampsAreStrictlyOrderedOnOneDevice(t *testing.T) {
	c, path, clock := syncCore(t)
	hlcs := []string{}
	task := add(t, c, core.AddTaskInput{Title: "t"})
	record := func() { hlcs = append(hlcs, stampMap(t, path, "task", task.UUID)["deadline"]) }

	for i, step := range []func(){
		func() { _ = c.Tasks.SetDeadline(task.ID, "2026-01-01") },                                 // clock unchanged
		func() { _ = c.Tasks.SetDeadline(task.ID, "2026-01-02") },                                 // clock unchanged again
		func() { clock.advance(-48 * time.Hour); _ = c.Tasks.SetDeadline(task.ID, "2026-01-03") }, // clock steps back
		func() { clock.advance(72 * time.Hour); _ = c.Tasks.SetDeadline(task.ID, "2026-01-04") },  // and forward
	} {
		step()
		record()
		if i > 0 && hlcs[i] <= hlcs[i-1] {
			t.Fatalf("stamp %d (%s) is not after %d (%s)", i, hlcs[i], i-1, hlcs[i-1])
		}
	}
	// The device id never changes.
	db := rawDB(t, path)
	defer db.Close()
	rows, _ := db.Query("SELECT DISTINCT device_id FROM field_versions")
	var ids []string
	for rows.Next() {
		var id string
		_ = rows.Scan(&id)
		ids = append(ids, id)
	}
	rows.Close()
	if len(ids) != 1 {
		t.Errorf("device ids = %v", ids)
	}
}

func TestLegacyDataGetsInitialStampsOnce(t *testing.T) {
	path := legacyDB(t, legacyV012)
	c := openAt(t, path)
	task, _ := c.Tasks.Get(5)
	got := stampMap(t, path, "task", task.UUID)
	if got["title"] == "" || got["when_kind"] == "" || got["sort_key"] == "" || got["status"] == "" {
		t.Errorf("legacy task unstamped: %v", got)
	}
	proj, _ := c.Projects.Get(3)
	if len(stampMap(t, path, "project", proj.UUID)) == 0 {
		t.Error("legacy project unstamped")
	}
	tags, _ := c.Tags.List()
	if m := stampMap(t, path, "task_tag", task.UUID+":"+tags[0].UUID); m["deleted_at"] == "" {
		t.Errorf("legacy tag membership unstamped: %v", m)
	}
	db := rawDB(t, path)
	for entity, want := range map[string]int{"habit_log": 3, "task_history": 2, "time_log": 2} { // the open log is device-local
		var n int
		_ = db.QueryRow("SELECT COUNT(DISTINCT uuid) FROM field_versions WHERE entity = ?", entity).Scan(&n)
		if n != want {
			t.Errorf("%s stamped %d rows, want %d", entity, n, want)
		}
	}
	db.Close()

	n := stampCount(t, path)
	before := stampMap(t, path, "task", task.UUID)
	openAt(t, path)
	openAt(t, path)
	if stampCount(t, path) != n {
		t.Error("reopening re-stamped existing rows")
	}
	if after := stampMap(t, path, "task", task.UUID); after["title"] != before["title"] {
		t.Error("initial stamps changed on reopen")
	}
	_ = sql.ErrNoRows
}
