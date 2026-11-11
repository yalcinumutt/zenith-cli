package core_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

func addSub(t *testing.T, c *core.Core, title string, parent int64) *models.Task {
	t.Helper()
	in := core.AddTaskInput{Title: title}
	if parent != 0 {
		in.ParentID = &parent
	}
	task, err := c.Tasks.Add(in)
	if err != nil {
		t.Fatal(err)
	}
	return task
}

func status(t *testing.T, c *core.Core, id int64) string {
	t.Helper()
	task, err := c.Tasks.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	return task.Status
}

func TestParentWithOpenSubtasksCannotBeCompletedDirectly(t *testing.T) {
	c, _ := newCore(t)
	parent := addSub(t, c, "parent", 0)
	addSub(t, c, "a", parent.ID)
	addSub(t, c, "b", parent.ID)

	err := c.Tasks.Complete(parent.ID)
	var oe *core.OpenSubtasksError
	if !errors.Is(err, core.ErrOpenSubtasks) || !errors.As(err, &oe) || oe.Count != 2 {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(err.Error(), "2 open subtasks") {
		t.Errorf("message = %q", err.Error())
	}
	if status(t, c, parent.ID) != models.StatusOpen {
		t.Error("parent changed despite refusal")
	}
}

func TestCompletingLastSubtaskCompletesParent(t *testing.T) {
	c, _ := newCore(t)
	parent := addSub(t, c, "parent", 0)
	a := addSub(t, c, "a", parent.ID)
	b := addSub(t, c, "b", parent.ID)

	if err := c.Tasks.Complete(a.ID); err != nil {
		t.Fatal(err)
	}
	if status(t, c, parent.ID) != models.StatusOpen {
		t.Fatal("parent completed while b is still open")
	}
	if err := c.Tasks.Complete(b.ID); err != nil {
		t.Fatal(err)
	}
	if status(t, c, parent.ID) != models.StatusCompleted {
		t.Fatal("parent should complete with its last subtask")
	}
	// Same completion path as manual: it leaves a status_changed entry.
	if got := actions(t, c, parent.ID); got[len(got)-1] != "status_changed" {
		t.Errorf("parent history = %v", got)
	}
}

func TestReopeningASubtaskReopensCompletedParents(t *testing.T) {
	c, _ := newCore(t)
	top := addSub(t, c, "top", 0)
	mid := addSub(t, c, "mid", top.ID)
	leaf := addSub(t, c, "leaf", mid.ID)

	if err := c.Tasks.Complete(leaf.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{leaf.ID, mid.ID, top.ID} {
		if status(t, c, id) != models.StatusCompleted {
			t.Fatalf("task %d should be completed (auto-complete walks up)", id)
		}
	}
	if err := c.Tasks.Reopen(leaf.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{leaf.ID, mid.ID, top.ID} {
		if status(t, c, id) != models.StatusOpen {
			t.Errorf("task %d should be open again", id)
		}
	}
}

func TestAddingASubtaskReopensACompletedParent(t *testing.T) {
	c, _ := newCore(t)
	parent := addSub(t, c, "parent", 0)
	_ = c.Tasks.Complete(parent.ID)
	addSub(t, c, "late", parent.ID)
	if status(t, c, parent.ID) != models.StatusOpen {
		t.Error("completed parent should reopen when it gains an open subtask")
	}
}

func TestSubtaskInheritsProjectAndIsAFullTask(t *testing.T) {
	c, _ := newCore(t)
	proj := int64(1)
	parent, _ := c.Tasks.Add(core.AddTaskInput{Title: "p", ProjectID: &proj})
	child := addSub(t, c, "child", parent.ID)
	got, _ := c.Tasks.Get(child.ID)
	if got.ProjectID == nil || *got.ProjectID != proj || got.ParentID == nil || *got.ParentID != parent.ID {
		t.Errorf("child = %+v", got)
	}
	if err := c.Tasks.Tag(child.ID, "own-tag", ""); err != nil {
		t.Fatal(err)
	}
	if err := c.Tasks.StartTimer(child.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(child.ID); !got.InProgress() || len(got.Tags) != 1 {
		t.Errorf("subtask should carry its own tags and timer: %+v", got)
	}
	kids, err := c.Tasks.Subtasks(parent.ID)
	if err != nil || len(kids) != 1 || kids[0].ID != child.ID {
		t.Errorf("Subtasks = %+v, %v", kids, err)
	}
}

func TestCyclesAreRejected(t *testing.T) {
	c, _ := newCore(t)
	a := addSub(t, c, "a", 0)
	b := addSub(t, c, "b", a.ID)
	d := addSub(t, c, "d", b.ID)

	for name, tc := range map[string]struct{ id, parent int64 }{
		"self":        {a.ID, a.ID},
		"child":       {a.ID, b.ID},
		"grandchild":  {a.ID, d.ID},
		"middle→leaf": {b.ID, d.ID},
	} {
		err := c.Tasks.SetParent(tc.id, &tc.parent)
		if !errors.Is(err, core.ErrCycle) {
			t.Errorf("%s: err = %v, want ErrCycle", name, err)
		}
	}
	if p, _ := c.Tasks.Get(a.ID); p.ParentID != nil {
		t.Error("rejected nesting must not change the task")
	}
	// A legal move still works.
	e := addSub(t, c, "e", 0)
	if err := c.Tasks.SetParent(d.ID, &e.ID); err != nil {
		t.Errorf("legal reparent: %v", err)
	}
}

func TestDeepNesting(t *testing.T) {
	c, _ := newCore(t)
	const depth = 60
	parent := int64(0)
	var ids []int64
	for i := 0; i < depth; i++ {
		task := addSub(t, c, "level", parent)
		ids = append(ids, task.ID)
		parent = task.ID
	}
	if err := c.Tasks.Complete(ids[0]); !errors.Is(err, core.ErrOpenSubtasks) {
		t.Fatalf("top should be blocked: %v", err)
	}
	if err := c.Tasks.Complete(ids[depth-1]); err != nil {
		t.Fatal(err)
	}
	for _, id := range ids {
		if status(t, c, id) != models.StatusCompleted {
			t.Fatalf("task %d not completed after the deepest leaf was", id)
		}
	}
	root := ids[0]
	if err := c.Tasks.SetParent(root, &ids[depth-1]); !errors.Is(err, core.ErrCycle) {
		t.Errorf("deep cycle not detected: %v", err)
	}
}

func TestCancelCascadesAndCountsAsClosed(t *testing.T) {
	c, _ := newCore(t)
	parent := addSub(t, c, "parent", 0)
	a := addSub(t, c, "a", parent.ID)
	b := addSub(t, c, "b", parent.ID)

	// Cancelling the only remaining open subtask after a completion completes the parent.
	_ = c.Tasks.Complete(a.ID)
	if err := c.Tasks.Cancel(b.ID); err != nil {
		t.Fatal(err)
	}
	if status(t, c, b.ID) != models.StatusCancelled || status(t, c, parent.ID) != models.StatusCompleted {
		t.Errorf("b=%s parent=%s", status(t, c, b.ID), status(t, c, parent.ID))
	}

	// Cancelling a parent cancels its open subtasks, not its completed ones.
	p2 := addSub(t, c, "p2", 0)
	x := addSub(t, c, "x", p2.ID)
	y := addSub(t, c, "y", p2.ID)
	_ = c.Tasks.Complete(x.ID)
	if err := c.Tasks.Cancel(p2.ID); err != nil {
		t.Fatal(err)
	}
	if status(t, c, p2.ID) != models.StatusCancelled || status(t, c, y.ID) != models.StatusCancelled || status(t, c, x.ID) != models.StatusCompleted {
		t.Errorf("p2=%s x=%s y=%s", status(t, c, p2.ID), status(t, c, x.ID), status(t, c, y.ID))
	}
	// Subtasks cannot be added to a cancelled task.
	if _, err := c.Tasks.Add(core.AddTaskInput{Title: "z", ParentID: &p2.ID}); !errors.Is(err, core.ErrInvalid) {
		t.Errorf("add under cancelled: %v", err)
	}
}

func TestDeleteRemovesSubtreeAndMayCompleteParent(t *testing.T) {
	c, _ := newCore(t)
	parent := addSub(t, c, "parent", 0)
	a := addSub(t, c, "a", parent.ID)
	b := addSub(t, c, "b", parent.ID)
	deep := addSub(t, c, "deep", b.ID)
	_ = c.Tasks.Complete(a.ID)

	if err := c.Tasks.Delete(b.ID); err != nil {
		t.Fatal(err)
	}
	for _, id := range []int64{b.ID, deep.ID} {
		if _, err := c.Tasks.Get(id); !errors.Is(err, core.ErrNotFound) {
			t.Errorf("task %d should be gone", id)
		}
	}
	if status(t, c, parent.ID) != models.StatusCompleted {
		t.Error("parent should complete once its last open subtask is removed")
	}
}

func TestNestingChangesAdjustTheOldParent(t *testing.T) {
	c, _ := newCore(t)
	p1 := addSub(t, c, "p1", 0)
	p2 := addSub(t, c, "p2", 0)
	a := addSub(t, c, "a", p1.ID)
	b := addSub(t, c, "b", p1.ID)
	_ = c.Tasks.Complete(a.ID)

	// Moving the last open subtask away completes the old parent.
	if err := c.Tasks.SetParent(b.ID, &p2.ID); err != nil {
		t.Fatal(err)
	}
	if status(t, c, p1.ID) != models.StatusCompleted {
		t.Error("old parent should complete")
	}
	// Moving an open task under a completed parent reopens it.
	c1 := addSub(t, c, "c", 0)
	if err := c.Tasks.SetParent(c1.ID, &p1.ID); err != nil {
		t.Fatal(err)
	}
	if status(t, c, p1.ID) != models.StatusOpen {
		t.Error("completed parent should reopen when an open task is nested under it")
	}
}

func TestSubtaskMutationsPublishOneEventBatch(t *testing.T) {
	c, _ := newCore(t)
	parent := addSub(t, c, "parent", 0)
	a := addSub(t, c, "a", parent.ID)
	events := collect(c)
	_ = c.Tasks.Complete(a.ID)

	var changed []int64
	lists := 0
	for _, e := range events() {
		switch v := e.(type) {
		case core.TaskChanged:
			changed = append(changed, v.TaskID)
		case core.ListsInvalidated:
			lists++
		}
	}
	if len(changed) != 2 || lists != 1 {
		t.Errorf("TaskChanged for %v, ListsInvalidated x%d; want the subtask and its parent, once", changed, lists)
	}
}
