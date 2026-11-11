package core_test

import (
	"database/sql"
	"errors"
	"testing"

	"github.com/yalcinumutt/zenith-cli/internal/core"
)

func TestTagsAreUniqueByCaseInsensitiveName(t *testing.T) {
	c, _ := newCore(t)
	a, _ := c.Tasks.Add(core.AddTaskInput{Title: "a"})
	b, _ := c.Tasks.Add(core.AddTaskInput{Title: "b"})

	if err := c.Tasks.Tag(a.ID, "Work", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.Tasks.Tag(b.ID, "work", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.Tasks.Tag(b.ID, "  WORK ", ""); err != nil { // already attached: no-op
		t.Fatal(err)
	}
	tags, _ := c.Tags.List()
	if len(tags) != 1 || tags[0].Name != "Work" {
		t.Fatalf("tags = %+v, want the single original 'Work'", tags)
	}
	got, _ := c.Tasks.Get(b.ID)
	if len(got.Tags) != 1 || got.Tags[0].ID != tags[0].ID {
		t.Errorf("task b tags = %+v", got.Tags)
	}
	if err := c.Tasks.Tag(a.ID, "  ", ""); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("blank tag: %v", err)
	}
}

func TestTagColorUpdatesOnlyWhenGiven(t *testing.T) {
	c, _ := newCore(t)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "t"})
	_ = c.Tasks.Tag(task.ID, "ui", "#111111")
	_ = c.Tasks.Tag(task.ID, "UI", "")
	if tags, _ := c.Tags.List(); tags[0].Color != "#111111" {
		t.Errorf("color = %q", tags[0].Color)
	}
	_ = c.Tasks.Tag(task.ID, "ui", "#222222")
	if tags, _ := c.Tags.List(); tags[0].Color != "#222222" {
		t.Errorf("color = %q", tags[0].Color)
	}
}

func TestUntag(t *testing.T) {
	c, _ := newCore(t)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "t"})
	other, _ := c.Tasks.Add(core.AddTaskInput{Title: "o"})
	_ = c.Tasks.Tag(task.ID, "Home", "")
	_ = c.Tasks.Tag(other.ID, "home", "")

	events := collect(c)
	if err := c.Tasks.Untag(task.ID, "HOME"); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(task.ID); len(got.Tags) != 0 {
		t.Errorf("tags = %+v", got.Tags)
	}
	if got, _ := c.Tasks.Get(other.ID); len(got.Tags) != 1 {
		t.Error("other task must keep the tag")
	}
	if len(events()) != 2 {
		t.Error("untag should publish TaskChanged + ListsInvalidated")
	}
	if err := c.Tasks.Untag(task.ID, "home"); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("untag twice: %v", err)
	}
	if err := c.Tasks.Untag(99, "home"); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("untag missing task: %v", err)
	}
}

func TestLegacyTagsSurviveDroppingTheUniqueConstraint(t *testing.T) {
	path := legacyDB(t, legacyV012)
	c := openAt(t, path)

	tags, _ := c.Tags.List()
	if len(tags) != 2 || tags[0].Name != "work" || tags[1].ID != 4 || tags[1].Name != "urgent" {
		t.Fatalf("tags = %+v", tags)
	}
	if got, _ := c.Tasks.Get(5); len(got.Tags) != 2 {
		t.Errorf("task 5 attachments lost: %+v", got.Tags)
	}
	for _, tag := range tags {
		if tag.UUID == "" {
			t.Errorf("tag %d lost its uuid", tag.ID)
		}
	}

	// The constraint is gone: the same name can exist twice at the storage level
	// (as after merging two devices), and the core resolves to the oldest.
	db, _ := sql.Open("sqlite", path)
	if _, err := db.Exec("INSERT INTO tags (uuid, name) VALUES ('dup', 'WORK')"); err != nil {
		t.Fatalf("duplicate name still rejected: %v", err)
	}
	db.Close()
	task, _ := c.Tasks.Get(7)
	_ = c.Tasks.Tag(task.ID, "Work", "")
	if got, _ := c.Tasks.Get(7); len(got.Tags) != 1 || got.Tags[0].ID != 1 {
		t.Errorf("should attach the oldest 'work' tag: %+v", got.Tags)
	}
	// Opening again does not rebuild or lose anything.
	c2 := openAt(t, path)
	if again, _ := c2.Tags.List(); len(again) != 3 {
		t.Errorf("tags after reopen = %d", len(again))
	}
}
