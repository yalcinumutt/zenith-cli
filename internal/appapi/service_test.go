package appapi

import (
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func newService(t *testing.T) (*Service, string) {
	t.Helper()
	dir := t.TempDir()
	st, err := storage.NewSQLiteStoreAtPath(filepath.Join(dir, "d.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	clock := fixedClock{time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	prefs := filepath.Join(dir, "prefs.json")
	return New(core.New(st, clock, nil), clock, prefs, "test"), prefs
}

func TestTodayAndListsMatchTheCore(t *testing.T) {
	s, _ := newService(t)
	mustAdd := func(in NewTask) TaskDTO {
		d, err := s.AddTask(in)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	mustAdd(NewTask{Title: "inbox"})
	today := mustAdd(NewTask{Title: "today", WhenKind: "scheduled", WhenDate: "2026-03-01", Notes: "n", Tags: []string{"Work", "work"}})
	mustAdd(NewTask{Title: "dinner", WhenKind: "scheduled", WhenDate: "2026-03-01", Evening: true})
	mustAdd(NewTask{Title: "later", WhenKind: "scheduled", WhenDate: "2026-03-09", Deadline: "2026-02-01"}) // overdue deadline -> Today

	td, err := s.Today()
	if err != nil {
		t.Fatal(err)
	}
	if len(td.Day) != 2 || len(td.Evening) != 1 || td.Evening[0].Title != "dinner" {
		t.Errorf("today = %+v", td)
	}
	if today.Notes != "n" || len(today.Tags) != 1 || today.Tags[0].Name != "Work" {
		t.Errorf("today task = %+v", today)
	}
	var overdue bool
	for _, d := range td.Day {
		overdue = overdue || (d.Title == "later" && d.Overdue)
	}
	if !overdue {
		t.Error("the overdue deadline task should be flagged")
	}
	inbox, _ := s.List("inbox")
	if len(inbox) != 1 || inbox[0].Title != "inbox" {
		t.Errorf("inbox = %+v", inbox)
	}
	c, _ := s.Counts()
	if c.Inbox != 1 || c.Today != 3 || c.Upcoming != 0 {
		t.Errorf("counts = %+v", c)
	}
	if _, err := s.List("nope"); err == nil {
		t.Error("unknown list accepted")
	}
}

func TestCompletionRefusalMessageAndSubtaskCounts(t *testing.T) {
	s, _ := newService(t)
	parent, _ := s.AddTask(NewTask{Title: "parent"})
	kid, _ := s.AddTask(NewTask{Title: "kid", ParentID: &parent.ID})

	err := s.Complete(parent.ID)
	if err == nil || !strings.Contains(err.Error(), "1 open subtask") {
		t.Fatalf("err = %v", err)
	}
	got, _ := s.GetTask(parent.ID)
	if got.SubtasksOpen != 1 || got.SubtasksTotal != 1 {
		t.Errorf("counts = %d/%d", got.SubtasksOpen, got.SubtasksTotal)
	}
	if err := s.Complete(kid.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetTask(parent.ID); got.Status != "completed" || got.SubtasksOpen != 0 {
		t.Errorf("parent = %+v", got)
	}
	if err := s.Reopen(kid.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.GetTask(parent.ID); got.Status != "open" {
		t.Errorf("parent should reopen: %+v", got)
	}
	kids, _ := s.Subtasks(parent.ID)
	if len(kids) != 1 || kids[0].ID != kid.ID {
		t.Errorf("subtasks = %+v", kids)
	}
}

func TestUpdatePatchTouchesOnlySetFields(t *testing.T) {
	s, _ := newService(t)
	proj, _ := s.AddProject("Site", nil)
	task, _ := s.AddTask(NewTask{Title: "t", ProjectID: &proj.ID, Deadline: "2026-04-04"})

	title := "renamed"
	got, err := s.UpdateTask(task.ID, TaskPatch{Title: &title})
	if err != nil || got.Title != "renamed" || got.Deadline != "2026-04-04" || got.ProjectName != "Site" {
		t.Fatalf("got %+v, %v", got, err)
	}
	kind, date, none := "scheduled", "2026-03-02", ""
	got, _ = s.UpdateTask(task.ID, TaskPatch{WhenKind: &kind, WhenDate: &date, Deadline: &none})
	if got.WhenKind != "scheduled" || got.WhenDate != "2026-03-02" || got.Deadline != "" {
		t.Errorf("got %+v", got)
	}
	tags := []string{"a", "b"}
	got, _ = s.UpdateTask(task.ID, TaskPatch{Tags: &tags})
	if len(got.Tags) != 2 {
		t.Errorf("tags = %+v", got.Tags)
	}
	got, _ = s.UpdateTask(task.ID, TaskPatch{ClearProject: true})
	if got.ProjectID != nil {
		t.Errorf("project not cleared: %+v", got)
	}
	empty := " "
	if _, err := s.UpdateTask(task.ID, TaskPatch{Title: &empty}); err == nil {
		t.Error("blank title accepted")
	}
	if _, err := s.AddTask(NewTask{Title: " "}); err == nil {
		t.Error("blank new task accepted")
	}
}

func TestSidebarAndProjectView(t *testing.T) {
	s, _ := newService(t)
	area, _ := s.AddArea("Work")
	inArea, _ := s.AddProject("Launch", &area.ID)
	solo, _ := s.AddProject("Side", nil)
	h, _ := s.AddHeading(inArea.ID, "Phase 1")
	_, _ = s.AddTask(NewTask{Title: "a", HeadingID: &h.ID})
	_, _ = s.AddTask(NewTask{Title: "b", ProjectID: &solo.ID})

	sb, err := s.Sidebar()
	if err != nil {
		t.Fatal(err)
	}
	if len(sb.Areas) != 1 || len(sb.Areas[0].Projects) != 1 || sb.Areas[0].Projects[0].OpenTasks != 1 || len(sb.Standalone) != 1 {
		t.Errorf("sidebar = %+v", sb)
	}
	view, err := s.ProjectView(inArea.ID)
	if err != nil || len(view.Headings) != 1 || len(view.Tasks) != 1 || view.Tasks[0].HeadingID == nil {
		t.Errorf("view = %+v, %v", view, err)
	}
	av, err := s.AreaView(area.ID)
	if err != nil || len(av.Area.Projects) != 1 {
		t.Errorf("area view = %+v, %v", av, err)
	}
	if err := s.DeleteProject(inArea.ID); err != nil {
		t.Fatal(err)
	}
	if sb, _ := s.Sidebar(); len(sb.Areas[0].Projects) != 0 {
		t.Error("deleted project still in the sidebar")
	}
}

func TestRepeatAndPrefsAndReminders(t *testing.T) {
	s, prefsPath := newService(t)
	task, _ := s.AddTask(NewTask{Title: "water", WhenKind: "scheduled", WhenDate: "2026-03-01"})
	got, err := s.SetRepeat(task.ID, RepeatDTO{Kind: "weeks", Interval: 2, Weekdays: []int{1, 3}})
	if err != nil || got.Repeat == nil || got.Repeat.Text != "Every 2 weeks on Mon, Wed" {
		t.Fatalf("repeat = %+v, %v", got.Repeat, err)
	}
	if err := s.Complete(task.ID); err != nil {
		t.Fatal(err)
	}
	up, _ := s.List("upcoming")
	if len(up) != 1 || up[0].Repeat == nil {
		t.Errorf("next occurrence = %+v", up)
	}
	log, _ := s.List("logbook")
	if len(log) != 1 || log[0].Status != "completed" {
		t.Errorf("logbook = %+v", log)
	}
	got, _ = s.RemoveRepeat(up[0].ID)
	if got.Repeat != nil {
		t.Error("repeat not removed")
	}

	if p := s.GetPrefs(); p.Theme != "system" || p.LastView != "today" {
		t.Errorf("defaults = %+v", p)
	}
	if _, err := s.SetPrefs(Prefs{Theme: "dark", LastView: "project:3"}); err != nil {
		t.Fatal(err)
	}
	s2 := New(nil, nil, prefsPath, "x")
	if p := s2.GetPrefs(); p.Theme != "dark" || p.LastView != "project:3" {
		t.Errorf("not persisted: %+v", p)
	}

	r, err := s.AddReminder(&task.ID, "ping", "2026-03-01T09:30:00Z")
	if err != nil || r.At != "2026-03-01T09:30:00Z" {
		t.Fatalf("reminder = %+v, %v", r, err)
	}
	if rs, _ := s.PendingReminders(); len(rs) != 1 {
		t.Errorf("pending = %+v", rs)
	}
	if _, err := s.AddReminder(nil, "x", "tomorrow"); err == nil {
		t.Error("bad time accepted")
	}
}

func TestChangeEventConversion(t *testing.T) {
	for _, e := range []core.Event{core.TaskChanged{TaskID: 4, Kind: core.ChangeStatus}, core.ListsInvalidated{Keys: []string{"tasks"}}, core.ReminderDue{ReminderID: 2}, core.ExternalChange{}} {
		if ev, ok := ToChangeEvent(e); !ok || ev.Type == "" || ev.Keys == nil {
			t.Errorf("%#v -> %+v, %v", e, ev, ok)
		}
	}
}
