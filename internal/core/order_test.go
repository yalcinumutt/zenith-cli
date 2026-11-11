package core_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

func order(t *testing.T, c *core.Core) []string {
	t.Helper()
	tasks, err := c.Tasks.List()
	if err != nil {
		t.Fatal(err)
	}
	return titles(tasks)
}

// snapshot returns id -> full row text for every task, read behind the core.
func snapshot(t *testing.T, path string) map[int64]string {
	t.Helper()
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	rows, err := db.Query("SELECT id, uuid, project_id, heading_id, title, status, when_kind, when_date, deadline, evening, sort_key, updated_at FROM tasks")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[int64]string{}
	for rows.Next() {
		var id int64
		cols := make([]sql.NullString, 11)
		ptrs := []any{&id}
		for i := range cols {
			ptrs = append(ptrs, &cols[i])
		}
		if err := rows.Scan(ptrs...); err != nil {
			t.Fatal(err)
		}
		s := ""
		for _, c := range cols {
			s += c.String + "|"
		}
		out[id] = s
	}
	return out
}

func TestNewTasksAppendInCreationOrder(t *testing.T) {
	c, _ := newCore(t)
	for _, n := range []string{"a", "b", "c", "d"} {
		add(t, c, core.AddTaskInput{Title: n})
	}
	if got := order(t, c); !reflect.DeepEqual(got, []string{"a", "b", "c", "d"}) {
		t.Errorf("order = %v", got)
	}
}

func TestReorderChangesOnlyTheMovedRow(t *testing.T) {
	path := filepath.Join(t.TempDir(), "o.db")
	c := openAt(t, path)
	ids := map[string]int64{}
	for _, n := range []string{"a", "b", "c", "d", "e"} {
		ids[n] = add(t, c, core.AddTaskInput{Title: n}).ID
	}
	before := snapshot(t, path)

	// Move "a" between "c" and "d".
	if err := c.Tasks.Reorder(ids["a"], ptr(ids["c"]), ptr(ids["d"])); err != nil {
		t.Fatal(err)
	}
	if got := order(t, c); !reflect.DeepEqual(got, []string{"b", "c", "a", "d", "e"}) {
		t.Fatalf("order = %v", got)
	}
	after := snapshot(t, path)
	for id, row := range before {
		if id == ids["a"] {
			if after[id] == row {
				t.Error("the moved row should have changed")
			}
			continue
		}
		if after[id] != row {
			t.Errorf("row %d was touched by a reorder:\n before %s\n after  %s", id, row, after[id])
		}
	}

	// To the very top and the very bottom.
	_ = c.Tasks.Reorder(ids["e"], nil, ptr(ids["b"]))
	_ = c.Tasks.Reorder(ids["e"], ptr(ids["d"]), nil)
	_ = c.Tasks.Reorder(ids["d"], nil, ptr(ids["b"]))
	if got := order(t, c); !reflect.DeepEqual(got, []string{"d", "b", "c", "a", "e"}) {
		t.Errorf("order = %v", got)
	}
	if err := c.Tasks.Reorder(ids["a"], ptr(999), nil); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("missing neighbour: %v", err)
	}
}

func TestRepeatedInsertionsAtTheSamePosition(t *testing.T) {
	c, _ := newCore(t)
	first := add(t, c, core.AddTaskInput{Title: "first"})
	last := add(t, c, core.AddTaskInput{Title: "last"})

	const n = 120
	var inserted []int64
	for i := 0; i < n; i++ {
		task := add(t, c, core.AddTaskInput{Title: "x"})
		// Always squeeze right after "first": its current successor is the upper neighbour.
		cur, _ := c.Tasks.List()
		if err := c.Tasks.Reorder(task.ID, ptr(first.ID), ptr(cur[1].ID)); err != nil {
			t.Fatal(err)
		}
		inserted = append(inserted, task.ID)
	}
	tasks, _ := c.Tasks.List()
	if len(tasks) != n+2 || tasks[0].ID != first.ID {
		t.Fatalf("first is not first: %d tasks", len(tasks))
	}
	// Each new task landed directly after "first", so they appear newest-first.
	for i := 0; i < n; i++ {
		if tasks[1+i].ID != inserted[n-1-i] {
			t.Fatalf("position %d holds task %d, want %d", i+1, tasks[1+i].ID, inserted[n-1-i])
		}
	}
	_ = last

	// And between two fixed neighbours, repeatedly.
	a := add(t, c, core.AddTaskInput{Title: "A"})
	b := add(t, c, core.AddTaskInput{Title: "B"})
	prev := a.ID
	for i := 0; i < 80; i++ {
		x := add(t, c, core.AddTaskInput{Title: "m"})
		if err := c.Tasks.Reorder(x.ID, ptr(prev), ptr(b.ID)); err != nil {
			t.Fatal(err)
		}
		prev = x.ID
	}
	all, _ := c.Tasks.List()
	var seenA, seenB int
	for i, tk := range all {
		if tk.ID == a.ID {
			seenA = i
		}
		if tk.ID == b.ID {
			seenB = i
		}
	}
	if seenB-seenA != 81 {
		t.Errorf("A..B span %d, want 81", seenB-seenA)
	}
}

func TestMovingAcrossHeadingsAndEveningIsOneWrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "h.db")
	c := openAt(t, path)
	p, _ := c.Projects.Add("P", nil)
	h1, _ := c.Headings.Add(p.ID, "H1")
	h2, _ := c.Headings.Add(p.ID, "H2")
	a := add(t, c, core.AddTaskInput{Title: "a", HeadingID: &h1.ID, When: sched("2026-03-01")})
	b := add(t, c, core.AddTaskInput{Title: "b", HeadingID: &h2.ID})
	other := add(t, c, core.AddTaskInput{Title: "other", HeadingID: &h2.ID})
	before := snapshot(t, path)

	// Drag "a" from H1 into H2, between b and other.
	if err := c.Tasks.Place(a.ID, core.Placement{AfterID: ptr(b.ID), BeforeID: ptr(other.ID), Heading: &core.HeadingChange{HeadingID: &h2.ID}}); err != nil {
		t.Fatal(err)
	}
	inH2, _ := c.Lists.Project(p.ID)
	var order2 []string
	for _, tk := range inH2 {
		if tk.HeadingID != nil && *tk.HeadingID == h2.ID {
			order2 = append(order2, tk.Title)
		}
	}
	if !reflect.DeepEqual(order2, []string{"b", "a", "other"}) {
		t.Errorf("H2 order = %v", order2)
	}
	after := snapshot(t, path)
	for id := range before {
		if id != a.ID && before[id] != after[id] {
			t.Errorf("row %d changed", id)
		}
	}

	// Day -> evening in Today.
	yes := true
	if err := c.Tasks.Place(a.ID, core.Placement{Evening: &yes}); err != nil {
		t.Fatal(err)
	}
	if today, _ := c.Lists.Today(); len(today.Evening) != 1 || today.Evening[0].ID != a.ID {
		t.Errorf("today = %+v", today)
	}
	// Leaving a heading, and the evening rule.
	if err := c.Tasks.Place(a.ID, core.Placement{Heading: &core.HeadingChange{}}); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(a.ID); got.HeadingID != nil || *got.ProjectID != p.ID {
		t.Errorf("a = %+v", got)
	}
	if err := c.Tasks.Place(b.ID, core.Placement{Evening: &yes}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("evening on unscheduled: %v", err)
	}
}

func TestHeadingsProjectsAndAreasReorder(t *testing.T) {
	c, _ := newCore(t)
	p, _ := c.Projects.Add("P", nil)
	h := []int64{}
	for _, n := range []string{"h1", "h2", "h3"} {
		x, _ := c.Headings.Add(p.ID, n)
		h = append(h, x.ID)
	}
	if err := c.Headings.Reorder(h[2], nil, ptr(h[0])); err != nil {
		t.Fatal(err)
	}
	hs, _ := c.Headings.List(p.ID)
	var got []string
	for _, x := range hs {
		got = append(got, x.Title)
	}
	if !reflect.DeepEqual(got, []string{"h3", "h1", "h2"}) {
		t.Errorf("headings = %v", got)
	}

	p2, _ := c.Projects.Add("P2", nil)
	if err := c.Projects.Reorder(p2.ID, nil, ptr(p.ID)); err != nil {
		t.Fatal(err)
	}
	if ps, _ := c.Projects.List(); ps[0].ID != p2.ID {
		t.Errorf("projects = %+v", ps)
	}
	a1, _ := c.Areas.Add("A1")
	a2, _ := c.Areas.Add("A2")
	if err := c.Areas.Reorder(a2.ID, nil, ptr(a1.ID)); err != nil {
		t.Fatal(err)
	}
	if as, _ := c.Areas.List(); as[0].ID != a2.ID {
		t.Errorf("areas = %+v", as)
	}
}

func TestLegacyRowsGetAStableInitialOrder(t *testing.T) {
	path := legacyDB(t, legacyV012)
	c := openAt(t, path)
	tasks, _ := c.Tasks.List()
	var ids []int64
	for _, tk := range tasks {
		ids = append(ids, tk.ID)
		if tk.SortKey == "" {
			t.Errorf("task %d has no sort key", tk.ID)
		}
	}
	if !reflect.DeepEqual(ids, []int64{1, 2, 5, 6, 7}) {
		t.Errorf("legacy order = %v, want id order", ids)
	}
	before := snapshot(t, path)
	// Reopening must not renumber anything.
	openAt(t, path)
	if !reflect.DeepEqual(before, snapshot(t, path)) {
		t.Error("reopen changed sort keys")
	}
	// New tasks go to the end; a legacy row can be squeezed between two others.
	n := add(t, c, core.AddTaskInput{Title: "new"})
	if err := c.Tasks.Reorder(n.ID, ptr(1), ptr(2)); err != nil {
		t.Fatal(err)
	}
	tasks, _ = c.Tasks.List()
	if tasks[1].ID != n.ID {
		t.Errorf("order after squeeze: %v", titles(tasks))
	}
	if p, _ := c.Projects.List(); len(p) != 1 || p[0].SortKey == "" {
		t.Errorf("projects = %+v", p)
	}
	_ = models.StatusOpen
}
