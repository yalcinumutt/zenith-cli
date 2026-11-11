package core_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

func sched(date string) models.When { return models.When{Kind: models.WhenScheduled, Date: date} }

func titles(tasks []models.Task) []string {
	out := []string{}
	for _, t := range tasks {
		out = append(out, t.Title)
	}
	return out
}

func list(t *testing.T, c *core.Core, kind core.ListKind) []string {
	t.Helper()
	tasks, err := c.Lists.Get(kind)
	if err != nil {
		t.Fatal(err)
	}
	return titles(tasks)
}

func add(t *testing.T, c *core.Core, in core.AddTaskInput) *models.Task {
	t.Helper()
	task, err := c.Tasks.Add(in)
	if err != nil {
		t.Fatalf("add %q: %v", in.Title, err)
	}
	return task
}

// clock is fixed at 2026-03-01 09:00 UTC by newCore.
func TestSixListsMembership(t *testing.T) {
	c, _ := newCore(t)
	proj, _ := c.Projects.Add("P", nil)
	area, _ := c.Areas.Add("A")

	add(t, c, core.AddTaskInput{Title: "inbox"})
	add(t, c, core.AddTaskInput{Title: "today", When: sched("2026-03-01")})
	add(t, c, core.AddTaskInput{Title: "overdue-when", When: sched("2026-02-20")})
	add(t, c, core.AddTaskInput{Title: "tomorrow", When: sched("2026-03-02")})
	add(t, c, core.AddTaskInput{Title: "later", When: sched("2026-04-01")})
	add(t, c, core.AddTaskInput{Title: "sooner", When: sched("2026-03-05")})
	add(t, c, core.AddTaskInput{Title: "someday", When: models.When{Kind: models.WhenSomeday}})
	add(t, c, core.AddTaskInput{Title: "in-project", ProjectID: &proj.ID})
	add(t, c, core.AddTaskInput{Title: "in-area", AreaID: &area.ID})
	add(t, c, core.AddTaskInput{Title: "deadline-today", Deadline: "2026-03-01"})
	add(t, c, core.AddTaskInput{Title: "deadline-overdue", Deadline: "2026-02-01", ProjectID: &proj.ID})
	add(t, c, core.AddTaskInput{Title: "deadline-future", Deadline: "2026-03-20"})
	add(t, c, core.AddTaskInput{Title: "someday-deadline-today", When: models.When{Kind: models.WhenSomeday}, Deadline: "2026-03-01"})
	done := add(t, c, core.AddTaskInput{Title: "done"})
	_ = c.Tasks.Complete(done.ID)
	gone := add(t, c, core.AddTaskInput{Title: "cancelled"})
	_ = c.Tasks.Cancel(gone.ID)

	want := map[core.ListKind][]string{
		core.ListInbox:    {"inbox", "deadline-future"},
		core.ListToday:    {"today", "overdue-when", "deadline-today", "deadline-overdue", "someday-deadline-today"},
		core.ListUpcoming: {"tomorrow", "sooner", "later"}, // by date, then order
		core.ListAnytime:  {"in-project", "in-area"},
		core.ListSomeday:  {"someday"},
		core.ListLogbook:  {"cancelled", "done"}, // newest first
	}
	for kind, titles := range want {
		got := list(t, c, kind)
		// Order within one list is by creation unless the list sorts; compare as sets where it does not.
		if kind == core.ListUpcoming || kind == core.ListLogbook {
			if !reflect.DeepEqual(got, titles) {
				t.Errorf("%s = %v, want %v", kind, got, titles)
			}
			continue
		}
		if !sameSet(got, titles) {
			t.Errorf("%s = %v, want %v", kind, got, titles)
		}
	}

	counts, _ := c.Lists.Counts()
	if counts.Inbox != 2 || counts.Today != 5 || counts.Upcoming != 3 {
		t.Errorf("counts = %+v", counts)
	}
}

func sameSet(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := map[string]int{}
	for _, x := range a {
		m[x]++
	}
	for _, x := range b {
		m[x]--
	}
	for _, v := range m {
		if v != 0 {
			return false
		}
	}
	return true
}

func TestEachTaskIsInExactlyOneOpenList(t *testing.T) {
	c, _ := newCore(t)
	proj, _ := c.Projects.Add("P", nil)
	for i, in := range []core.AddTaskInput{
		{Title: "a"}, {Title: "b", When: sched("2026-03-01")}, {Title: "c", When: sched("2026-09-09")},
		{Title: "d", Deadline: "2026-03-01"}, {Title: "e", ProjectID: &proj.ID},
		{Title: "f", When: models.When{Kind: models.WhenSomeday}, Deadline: "2020-01-01"},
	} {
		add(t, c, in)
		_ = i
	}
	seen := map[string]int{}
	for _, k := range []core.ListKind{core.ListInbox, core.ListToday, core.ListUpcoming, core.ListAnytime, core.ListSomeday} {
		for _, title := range list(t, c, k) {
			seen[title]++
		}
	}
	if len(seen) != 6 {
		t.Errorf("seen = %v", seen)
	}
	for title, n := range seen {
		if n != 1 {
			t.Errorf("%q appears in %d lists", title, n)
		}
	}
}

func TestDayRolloverKeepsUnfinishedTodayInToday(t *testing.T) {
	c, clock := newCore(t)
	add(t, c, core.AddTaskInput{Title: "tonight", When: sched("2026-03-01")})
	add(t, c, core.AddTaskInput{Title: "next", When: sched("2026-03-02")})

	if got := list(t, c, core.ListToday); !reflect.DeepEqual(got, []string{"tonight"}) {
		t.Fatalf("day 1 today = %v", got)
	}
	if got := list(t, c, core.ListUpcoming); !reflect.DeepEqual(got, []string{"next"}) {
		t.Fatalf("day 1 upcoming = %v", got)
	}

	clock.advance(24 * time.Hour) // 2026-03-02
	if got := list(t, c, core.ListToday); !sameSet(got, []string{"tonight", "next"}) {
		t.Errorf("day 2 today = %v (an unfinished Today task must stay)", got)
	}
	if got := list(t, c, core.ListUpcoming); len(got) != 0 {
		t.Errorf("day 2 upcoming = %v", got)
	}

	clock.advance(10 * 24 * time.Hour)
	if got := list(t, c, core.ListToday); len(got) != 2 {
		t.Errorf("day 12 today = %v", got)
	}
}

func TestDeadlineOverdueEntersTodayWithoutWhen(t *testing.T) {
	c, clock := newCore(t)
	add(t, c, core.AddTaskInput{Title: "due", Deadline: "2026-03-03"})
	if len(list(t, c, core.ListToday)) != 0 || len(list(t, c, core.ListInbox)) != 1 {
		t.Fatal("a future deadline alone keeps the task in the Inbox")
	}
	clock.advance(2 * 24 * time.Hour) // the deadline day
	if got := list(t, c, core.ListToday); !reflect.DeepEqual(got, []string{"due"}) {
		t.Errorf("deadline today: %v", got)
	}
	clock.advance(5 * 24 * time.Hour) // overdue
	if got := list(t, c, core.ListToday); !reflect.DeepEqual(got, []string{"due"}) {
		t.Errorf("overdue: %v", got)
	}
	if len(list(t, c, core.ListInbox)) != 0 {
		t.Error("an overdue task leaves the Inbox")
	}
}

func TestThisEveningSplit(t *testing.T) {
	c, _ := newCore(t)
	add(t, c, core.AddTaskInput{Title: "morning", When: sched("2026-03-01")})
	eve := add(t, c, core.AddTaskInput{Title: "dinner", When: sched("2026-03-01"), Evening: true})

	today, err := c.Lists.Today()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(titles(today.Day), []string{"morning"}) || !reflect.DeepEqual(titles(today.Evening), []string{"dinner"}) {
		t.Errorf("today = day %v evening %v", titles(today.Day), titles(today.Evening))
	}

	_ = c.Tasks.SetEvening(eve.ID, false)
	if today, _ := c.Lists.Today(); len(today.Evening) != 0 || len(today.Day) != 2 {
		t.Error("clearing evening moves it back to the day")
	}
	// Moving off a scheduled day clears the evening flag; unscheduled tasks cannot be evening.
	_ = c.Tasks.SetEvening(eve.ID, true)
	_ = c.Tasks.SetWhen(eve.ID, models.When{Kind: models.WhenSomeday})
	if got, _ := c.Tasks.Get(eve.ID); got.Evening {
		t.Error("evening should clear when the task is no longer scheduled")
	}
	if err := c.Tasks.SetEvening(eve.ID, true); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("evening on an unscheduled task: %v", err)
	}
	if _, err := c.Tasks.Add(core.AddTaskInput{Title: "x", Evening: true}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("add evening unscheduled: %v", err)
	}
}

func TestSubtasksOnlyAppearNestedAndInTheirProject(t *testing.T) {
	c, _ := newCore(t)
	proj, _ := c.Projects.Add("P", nil)
	parent := add(t, c, core.AddTaskInput{Title: "parent", When: sched("2026-03-01")})
	sub := add(t, c, core.AddTaskInput{Title: "sub", ParentID: &parent.ID, ProjectID: &proj.ID, When: sched("2026-03-01"), Deadline: "2020-01-01"})
	_ = sub

	for _, k := range core.ListKinds {
		for _, title := range list(t, c, k) {
			if title == "sub" {
				t.Errorf("subtask leaked into %s", k)
			}
		}
	}
	inProject, _ := c.Lists.Project(proj.ID)
	if !reflect.DeepEqual(titles(inProject), []string{"sub"}) {
		t.Errorf("project list = %v", titles(inProject))
	}
	kids, _ := c.Tasks.Subtasks(parent.ID)
	if len(kids) != 1 {
		t.Error("nested under its parent")
	}
}

func TestSetWhenAndDeadlineValidation(t *testing.T) {
	c, _ := newCore(t)
	task := add(t, c, core.AddTaskInput{Title: "t"})
	if err := c.Tasks.SetWhen(task.ID, models.When{Kind: models.WhenScheduled, Date: "soon"}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("bad date: %v", err)
	}
	if err := c.Tasks.SetDeadline(task.ID, "31/12/2026"); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("bad deadline: %v", err)
	}
	if err := c.Tasks.SetDeadline(task.ID, "2026-12-31"); err != nil {
		t.Fatal(err)
	}
	if err := c.Tasks.SetDeadline(task.ID, ""); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(task.ID); got.Deadline != "" {
		t.Errorf("deadline = %q", got.Deadline)
	}

	now := time.Date(2026, 3, 1, 23, 59, 0, 0, time.UTC)
	for in, want := range map[string]models.When{
		"today":      sched("2026-03-01"),
		"Tomorrow":   sched("2026-03-02"),
		"someday":    {Kind: models.WhenSomeday},
		"anytime":    {Kind: models.WhenAnytime},
		"2026-07-04": sched("2026-07-04"),
	} {
		if got, err := core.ParseWhen(in, now); err != nil || got != want {
			t.Errorf("ParseWhen(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	if _, err := core.ParseWhen("next week", now); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("ParseWhen junk: %v", err)
	}
}

func TestLegacyDatesMapToWhenAndDeadline(t *testing.T) {
	path := legacyDB(t, legacyV012)
	c := openAt(t, path)

	cases := map[int64]struct {
		when     models.When
		deadline string
	}{
		1: {sched("2026-03-02"), "2026-03-10"},
		5: {models.When{Kind: models.WhenAnytime}, "2026-03-05"},
		7: {sched("2026-03-04"), ""},
		2: {models.When{Kind: models.WhenAnytime}, ""},
	}
	for id, want := range cases {
		got, err := c.Tasks.Get(id)
		if err != nil || got.When != want.when || got.Deadline != want.deadline {
			t.Errorf("task %d: when %+v deadline %q (err %v), want %+v / %q", id, got.When, got.Deadline, err, want.when, want.deadline)
		}
	}
	// The free-form columns are gone and a second open is a no-op.
	db := rawDB(t, path)
	defer db.Close()
	for _, col := range []string{"planned_date", "due_date"} {
		var n int
		_ = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = ?", col).Scan(&n)
		if n != 0 {
			t.Errorf("column %s still exists", col)
		}
	}
	c2 := openAt(t, path)
	if got, _ := c2.Tasks.Get(1); got.When != sched("2026-03-02") || got.Deadline != "2026-03-10" {
		t.Errorf("after reopen: %+v %q", got.When, got.Deadline)
	}
}
