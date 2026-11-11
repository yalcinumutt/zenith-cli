package core_test

import (
	"errors"
	"testing"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

func TestProjects(t *testing.T) {
	c, _ := newCore(t)
	events := collect(c)

	p, err := c.Projects.Add("Website", nil)
	if err != nil || p.ID == 0 || p.Name != "Website" {
		t.Fatalf("Add = %+v, %v", p, err)
	}
	list, err := c.Projects.List()
	if err != nil {
		t.Fatal(err)
	}
	// A fresh database has no seeded "Anytime" project any more.
	if len(list) != 1 || list[0].Name != "Website" || list[0].Status != models.StatusOpen {
		t.Errorf("projects = %+v", list)
	}
	got := events()
	if len(got) != 1 {
		t.Fatalf("events = %v", got)
	}
	if li, ok := got[0].(core.ListsInvalidated); !ok || li.Keys[0] != core.ListProjects {
		t.Errorf("event = %#v", got[0])
	}
}

func TestHabits(t *testing.T) {
	c, _ := newCore(t)
	h, err := c.Habits.Add("Read")
	if err != nil || h.Frequency != core.DefaultHabitFrequency {
		t.Fatalf("Add = %+v, %v", h, err)
	}
	events := collect(c)
	if err := c.Habits.Log(h.ID); err != nil {
		t.Fatal(err)
	}
	if got := events(); len(got) != 1 {
		t.Errorf("events after log = %v", got)
	}
	list, _ := c.Habits.List()
	if len(list) != 1 || list[0].Name != "Read" {
		t.Errorf("list = %+v", list)
	}

	err = c.Habits.Log(99)
	if !errors.Is(err, core.ErrNotFound) || err.Error() != "habit with ID 99 not found" {
		t.Errorf("Log(99) err = %v", err)
	}
	if got := events(); len(got) != 0 {
		t.Errorf("failed log published %v", got)
	}
}

func TestSearchAndOverview(t *testing.T) {
	c, _ := newCore(t)
	_, _ = c.Tasks.Add(core.AddTaskInput{Title: "Buy milk"})
	_, _ = c.Tasks.Add(core.AddTaskInput{Title: "Write report"})
	_, _ = c.Habits.Add("Drink milk")
	_, _ = c.Habits.Add("Stretch")

	res, err := c.Search("milk")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Tasks) != 1 || res.Tasks[0].Title != "Buy milk" {
		t.Errorf("tasks = %+v", res.Tasks)
	}
	if len(res.Habits) != 1 || res.Habits[0].Name != "Drink milk" {
		t.Errorf("habits = %+v", res.Habits)
	}
	if res, _ := c.Search("zzz"); len(res.Tasks)+len(res.Habits) != 0 {
		t.Errorf("expected no matches, got %+v", res)
	}

	o, err := c.Overview()
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Tasks) != 2 || len(o.Habits) != 2 || len(o.Projects) != 0 {
		t.Errorf("overview = %d tasks, %d habits, %d projects", len(o.Tasks), len(o.Habits), len(o.Projects))
	}
}
