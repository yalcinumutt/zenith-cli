package core_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

func newArea(t *testing.T, c *core.Core, name string) *models.Area {
	t.Helper()
	a, err := c.Areas.Add(name)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func newProject(t *testing.T, c *core.Core, name string, area *int64) *models.Project {
	t.Helper()
	p, err := c.Projects.Add(name, area)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLegacyAnytimeProjectIsRetired(t *testing.T) {
	path := legacyDB(t, legacyV012)
	c := openAt(t, path)

	projects, _ := c.Projects.List()
	for _, p := range projects {
		if p.Name == "Anytime" {
			t.Fatalf("Anytime project survived: %+v", p)
		}
	}
	// Tasks 1 and 6 were in project 1 (Anytime); 2 and 5 were in project 3; 7 had none.
	want := map[int64]*int64{1: nil, 6: nil, 7: nil}
	for id := range want {
		task, _ := c.Tasks.Get(id)
		if task.ProjectID != nil {
			t.Errorf("task %d should be project-less, has project %d", id, *task.ProjectID)
		}
	}
	for _, id := range []int64{2, 5} {
		task, _ := c.Tasks.Get(id)
		if task.ProjectID == nil || *task.ProjectID != 3 {
			t.Errorf("task %d should stay in project 3", id)
		}
	}
	if p, _ := c.Projects.Get(3); p.Status != models.StatusOpen || p.AreaID != nil {
		t.Errorf("migrated project 3 = %+v", p)
	}

	// Re-opening never recreates it.
	c2 := openAt(t, path)
	again, _ := c2.Projects.List()
	if len(again) != 1 || again[0].ID != 3 {
		t.Errorf("projects after reopen = %+v", again)
	}
}

func TestUsersOwnProjectNamedAnytimeIsKept(t *testing.T) {
	path := legacyDB(t, legacyV012)
	db, _ := sql.Open("sqlite", path)
	_, err := db.Exec("INSERT INTO projects (id, name, description) VALUES (9, 'Anytime', 'my own list')")
	db.Close()
	if err != nil {
		t.Fatal(err)
	}
	c := openAt(t, path)
	if _, err := c.Projects.Get(9); err != nil {
		t.Errorf("a user's own 'Anytime' project must survive: %v", err)
	}
}

func TestAreaProjectHeadingCrud(t *testing.T) {
	c, _ := newCore(t)
	area := newArea(t, c, "Work")
	proj := newProject(t, c, "Launch", &area.ID)
	standalone := newProject(t, c, "Side", nil)
	h, err := c.Headings.Add(proj.ID, "Phase 1")
	if err != nil {
		t.Fatal(err)
	}

	if err := c.Areas.Rename(area.ID, "Job"); err != nil {
		t.Fatal(err)
	}
	if err := c.Projects.Rename(proj.ID, "Launch v2"); err != nil {
		t.Fatal(err)
	}
	if err := c.Headings.Rename(h.ID, "Phase one"); err != nil {
		t.Fatal(err)
	}
	if a, _ := c.Areas.Get(area.ID); a.Name != "Job" {
		t.Errorf("area = %+v", a)
	}
	if p, _ := c.Projects.Get(proj.ID); p.Name != "Launch v2" || p.AreaID == nil || *p.AreaID != area.ID {
		t.Errorf("project = %+v", p)
	}
	if hs, _ := c.Headings.List(proj.ID); len(hs) != 1 || hs[0].Title != "Phase one" {
		t.Errorf("headings = %+v", hs)
	}

	// Move a project between an area and standalone.
	if err := c.Projects.Move(standalone.ID, &area.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.Projects.Move(proj.ID, nil); err != nil {
		t.Fatal(err)
	}
	if p, _ := c.Projects.Get(proj.ID); p.AreaID != nil {
		t.Error("project should be standalone")
	}
	if p, _ := c.Projects.Get(standalone.ID); p.AreaID == nil || *p.AreaID != area.ID {
		t.Error("project should be in the area")
	}

	// Validation.
	if _, err := c.Areas.Add("  "); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("empty area name: %v", err)
	}
	if _, err := c.Projects.Add("x", ptr(999)); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("project in missing area: %v", err)
	}
	if _, err := c.Headings.Add(999, "x"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("heading in missing project: %v", err)
	}
	if err := c.Projects.Move(proj.ID, ptr(999)); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("move to missing area: %v", err)
	}
}

func ptr(v int64) *int64 { return &v }

func TestTaskContainerRules(t *testing.T) {
	c, _ := newCore(t)
	area := newArea(t, c, "Home")
	p1 := newProject(t, c, "P1", nil)
	p2 := newProject(t, c, "P2", nil)
	h1, _ := c.Headings.Add(p1.ID, "H1")

	// A heading implies its project.
	task, err := c.Tasks.Add(core.AddTaskInput{Title: "t", HeadingID: &h1.ID})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(task.ID); got.ProjectID == nil || *got.ProjectID != p1.ID || got.HeadingID == nil {
		t.Errorf("task = %+v", got)
	}
	// ...and must agree with an explicit project.
	if _, err := c.Tasks.Add(core.AddTaskInput{Title: "bad", HeadingID: &h1.ID, ProjectID: &p2.ID}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("heading/project mismatch: %v", err)
	}
	// A task is in a project or an area, never both.
	if _, err := c.Tasks.Add(core.AddTaskInput{Title: "bad", ProjectID: &p1.ID, AreaID: &area.ID}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("project+area: %v", err)
	}
	if _, err := c.Tasks.Add(core.AddTaskInput{Title: "bad", ProjectID: ptr(999)}); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("missing project: %v", err)
	}

	// Moving to another project drops the old heading.
	if err := c.Tasks.MoveToProject(task.ID, p2.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(task.ID); *got.ProjectID != p2.ID || got.HeadingID != nil {
		t.Errorf("after move = %+v", got)
	}
	// Setting a heading re-files the task into that heading's project.
	if err := c.Tasks.SetHeading(task.ID, &h1.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(task.ID); *got.ProjectID != p1.ID || *got.HeadingID != h1.ID {
		t.Errorf("after heading = %+v", got)
	}
	// Filing in an area leaves the project and heading.
	if err := c.Tasks.FileInArea(task.ID, area.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(task.ID); got.ProjectID != nil || got.HeadingID != nil || *got.AreaID != area.ID {
		t.Errorf("after area = %+v", got)
	}
	if err := c.Tasks.Unfile(task.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(task.ID); got.ProjectID != nil || got.AreaID != nil || got.HeadingID != nil {
		t.Errorf("after unfile = %+v", got)
	}
}

func TestProjectCompletionRules(t *testing.T) {
	c, _ := newCore(t)
	p := newProject(t, c, "P", nil)
	a, _ := c.Tasks.Add(core.AddTaskInput{Title: "a", ProjectID: &p.ID})
	b, _ := c.Tasks.Add(core.AddTaskInput{Title: "b", ProjectID: &p.ID})

	err := c.Projects.Complete(p.ID)
	var oe *core.OpenTasksError
	if !errors.Is(err, core.ErrOpenTasks) || !errors.As(err, &oe) || oe.Count != 2 {
		t.Fatalf("complete with open tasks: %v", err)
	}
	_ = c.Tasks.Complete(a.ID)
	_ = c.Tasks.Complete(b.ID)
	if err := c.Projects.Complete(p.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Projects.Get(p.ID); got.Status != models.StatusCompleted {
		t.Errorf("status = %s", got.Status)
	}
	if err := c.Projects.Reopen(p.ID); err != nil {
		t.Fatal(err)
	}

	// Cancelling cascades to open tasks and their subtasks.
	c1, _ := c.Tasks.Add(core.AddTaskInput{Title: "c", ProjectID: &p.ID})
	sub, _ := c.Tasks.Add(core.AddTaskInput{Title: "sub", ParentID: &c1.ID})
	if err := c.Projects.Cancel(p.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Projects.Get(p.ID); got.Status != models.StatusCancelled {
		t.Errorf("project = %s", got.Status)
	}
	for _, id := range []int64{c1.ID, sub.ID} {
		if status(t, c, id) != models.StatusCancelled {
			t.Errorf("task %d should be cancelled", id)
		}
	}
	if status(t, c, a.ID) != models.StatusCompleted {
		t.Error("completed tasks stay completed")
	}
}

// countRows counts rows in a table behind the core's back.
func countRows(t *testing.T, path, table string) int {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var n int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestDeletingAProjectLeavesNothingBehind(t *testing.T) {
	path := filepath.Join(t.TempDir(), "d.db")
	c := openAt(t, path)
	p := newProject(t, c, "Doomed", nil)
	other := newProject(t, c, "Safe", nil)
	h, _ := c.Headings.Add(p.ID, "Section")

	t1, _ := c.Tasks.Add(core.AddTaskInput{Title: "in heading", HeadingID: &h.ID})
	t2, _ := c.Tasks.Add(core.AddTaskInput{Title: "in project", ProjectID: &p.ID})
	sub, _ := c.Tasks.Add(core.AddTaskInput{Title: "sub", ParentID: &t2.ID})
	keep, _ := c.Tasks.Add(core.AddTaskInput{Title: "keep", ProjectID: &other.ID})
	// Dependent rows: tags, time logs, history, reminders.
	_ = c.Tasks.Tag(t1.ID, "x", "")
	_ = c.Tasks.StartTimer(sub.ID)
	_, _ = c.Reminders.Add("r", 1<<40, &t2.ID)

	if err := c.Projects.Delete(p.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{t1.ID, t2.ID, sub.ID} {
		if _, err := c.Tasks.Get(id); !errors.Is(err, core.ErrNotFound) {
			t.Errorf("task %d should be gone: %v", id, err)
		}
	}
	if _, err := c.Tasks.Get(keep.ID); err != nil {
		t.Errorf("task in another project must survive: %v", err)
	}
	if _, err := c.Headings.Get(h.ID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("heading should be gone: %v", err)
	}
	if _, err := c.Projects.Get(p.ID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("project should be gone: %v", err)
	}

	// Only the surviving task's rows remain anywhere.
	for table, want := range map[string]int{"tasks": 1, "headings": 0, "task_tags": 0, "task_time_logs": 0, "scheduled_timers": 0} {
		if got := countRows(t, path, table); got != want {
			t.Errorf("%s rows = %d, want %d", table, got, want)
		}
	}
	if got := countRows(t, path, "task_history"); got != 1 { // just "created" for the kept task
		t.Errorf("task_history rows = %d, want 1", got)
	}
}

func TestDeletingAHeadingRemovesItsTasksOnly(t *testing.T) {
	c, _ := newCore(t)
	p := newProject(t, c, "P", nil)
	h1, _ := c.Headings.Add(p.ID, "H1")
	h2, _ := c.Headings.Add(p.ID, "H2")
	a, _ := c.Tasks.Add(core.AddTaskInput{Title: "a", HeadingID: &h1.ID})
	b, _ := c.Tasks.Add(core.AddTaskInput{Title: "b", HeadingID: &h2.ID})
	loose, _ := c.Tasks.Add(core.AddTaskInput{Title: "loose", ProjectID: &p.ID})

	if err := c.Headings.Delete(h1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Tasks.Get(a.ID); !errors.Is(err, core.ErrNotFound) {
		t.Error("task under the deleted heading should be gone")
	}
	for _, id := range []int64{b.ID, loose.ID} {
		if _, err := c.Tasks.Get(id); err != nil {
			t.Errorf("task %d should remain: %v", id, err)
		}
	}
}

func TestMovingAHeadingTakesItsTasks(t *testing.T) {
	c, _ := newCore(t)
	p1 := newProject(t, c, "P1", nil)
	p2 := newProject(t, c, "P2", nil)
	h, _ := c.Headings.Add(p1.ID, "H")
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "t", HeadingID: &h.ID})

	if err := c.Headings.Move(h.ID, p2.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Headings.Get(h.ID); got.ProjectID != p2.ID {
		t.Errorf("heading = %+v", got)
	}
	if got, _ := c.Tasks.Get(task.ID); *got.ProjectID != p2.ID || *got.HeadingID != h.ID {
		t.Errorf("task = %+v", got)
	}
	if err := c.Headings.Move(h.ID, 999); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("move to missing project: %v", err)
	}
}

func TestDeletingAnAreaKeepsItsContents(t *testing.T) {
	c, _ := newCore(t)
	area := newArea(t, c, "A")
	p := newProject(t, c, "P", &area.ID)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "t", AreaID: &area.ID})

	if err := c.Areas.Delete(area.ID); err != nil {
		t.Fatal(err)
	}
	if got, err := c.Projects.Get(p.ID); err != nil || got.AreaID != nil {
		t.Errorf("project should survive standalone: %+v, %v", got, err)
	}
	if got, err := c.Tasks.Get(task.ID); err != nil || got.AreaID != nil {
		t.Errorf("task should survive unfiled: %+v, %v", got, err)
	}
	if _, err := c.Areas.Get(area.ID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("area should be gone: %v", err)
	}
}
