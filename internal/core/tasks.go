package core

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// TaskService owns the task rules: status transitions, history entries, timer
// handling and not-found errors.
type TaskService struct {
	store    storage.Store
	clock    Clock
	notifier Notifier
	bus      *bus
}

// AddTaskInput is what a front-end supplies to create a task.
type AddTaskInput struct {
	Title     string
	ProjectID *int64 // a task is filed in a project or an area, never both
	AreaID    *int64
	HeadingID *int64 // implies the heading's project
	ParentID  *int64 // makes the new task a Subtask of this task
	DueDate   *time.Time
	Priority  models.Priority
	Recurring string
}

// changes collects what a mutation touched so events publish only after the
// transaction commits.
type changes struct {
	items []TaskChanged
}

func (c *changes) add(id int64, kind ChangeKind) { c.items = append(c.items, TaskChanged{id, kind}) }

// run executes fn in one transaction and announces its changes after commit.
func (s *TaskService) run(fn func(tx storage.Store, ch *changes) error) error {
	ch := &changes{}
	if err := s.store.WithTx(func(tx storage.Store) error { return fn(tx, ch) }); err != nil {
		return err
	}
	for _, c := range ch.items {
		s.bus.publish(c)
	}
	if len(ch.items) > 0 {
		s.bus.publish(ListsInvalidated{Keys: []string{ListTasks}})
	}
	return nil
}

func (s *TaskService) Add(in AddTaskInput) (*models.Task, error) {
	task := &models.Task{
		Title:     in.Title,
		Status:    models.StatusOpen,
		Priority:  in.Priority,
		ProjectID: in.ProjectID,
		AreaID:    in.AreaID,
		HeadingID: in.HeadingID,
		ParentID:  in.ParentID,
		DueDate:   in.DueDate,
		Recurring: in.Recurring,
	}
	err := s.run(func(tx storage.Store, ch *changes) error {
		if in.ParentID != nil {
			parent, err := s.get(tx, *in.ParentID)
			if err != nil {
				return err
			}
			if parent.Status == models.StatusCancelled {
				return invalidError(fmt.Sprintf("task %d is cancelled; reopen it before adding subtasks", parent.ID))
			}
			if task.ProjectID == nil && task.AreaID == nil && task.HeadingID == nil {
				task.ProjectID = parent.ProjectID // a Subtask files with its parent unless told otherwise
			}
			// A new open Subtask makes a completed parent open work again.
			if parent.Status == models.StatusCompleted {
				if err := s.reopen(tx, ch, parent.ID); err != nil {
					return err
				}
			}
		}
		if err := s.normalizeContainer(tx, task); err != nil {
			return err
		}
		if err := tx.AddTask(task); err != nil {
			return err
		}
		ch.add(task.ID, ChangeCreated)
		return s.record(tx, task.ID, "created", fmt.Sprintf("Task created: '%s'", task.Title))
	})
	if err != nil {
		return nil, err
	}
	return task, nil
}

func (s *TaskService) List() ([]models.Task, error) {
	return s.store.GetTasks()
}

// Get returns the task or a *NotFoundError.
func (s *TaskService) Get(id int64) (*models.Task, error) {
	return s.get(s.store, id)
}

// Subtasks returns the direct Subtasks of a task, oldest first.
func (s *TaskService) Subtasks(id int64) ([]models.Task, error) {
	if _, err := s.get(s.store, id); err != nil {
		return nil, err
	}
	return s.store.GetSubtasks(id)
}

// Complete marks the task completed and stops its running timer. It is refused
// while the task has open Subtasks. Completing the last open Subtask of a task
// completes that task too, through this same path (ADR 0005).
func (s *TaskService) Complete(id int64) error {
	return s.run(func(tx storage.Store, ch *changes) error { return s.complete(tx, ch, id) })
}

// Reopen makes a completed or cancelled task open again. A completed parent of
// a reopened Subtask is reopened too.
func (s *TaskService) Reopen(id int64) error {
	return s.run(func(tx storage.Store, ch *changes) error { return s.reopen(tx, ch, id) })
}

// Cancel marks the task and its open Subtasks cancelled, stopping their timers.
func (s *TaskService) Cancel(id int64) error {
	return s.run(func(tx storage.Store, ch *changes) error { return s.cancel(tx, ch, id) })
}

func (s *TaskService) cancel(tx storage.Store, ch *changes, id int64) error {
	t, err := s.get(tx, id)
	if err != nil {
		return err
	}
	if t.Status == models.StatusCancelled {
		return nil
	}
	if err := s.cancelTree(tx, ch, t); err != nil {
		return err
	}
	if t.ParentID != nil {
		return s.completeParentIfDone(tx, ch, *t.ParentID)
	}
	return nil
}

// SetParent nests the task under parentID, or makes it top-level when nil. A
// nesting that would make a task its own ancestor is rejected with ErrCycle.
func (s *TaskService) SetParent(id int64, parentID *int64) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		t, err := s.get(tx, id)
		if err != nil {
			return err
		}
		if sameID(t.ParentID, parentID) {
			return nil
		}
		oldParent := t.ParentID
		if parentID != nil {
			if err := s.checkAcyclic(tx, id, *parentID); err != nil {
				return err
			}
			parent, err := s.get(tx, *parentID)
			if err != nil {
				return err
			}
			if parent.Status == models.StatusCancelled {
				return invalidError(fmt.Sprintf("task %d is cancelled; reopen it before adding subtasks", parent.ID))
			}
			if parent.Status == models.StatusCompleted && t.Status == models.StatusOpen {
				if err := s.reopen(tx, ch, parent.ID); err != nil {
					return err
				}
			}
		}
		if err := s.save(tx, t, func(t *models.Task) { t.ParentID = parentID }); err != nil {
			return err
		}
		ch.add(id, ChangeUpdated)
		if oldParent != nil {
			return s.completeParentIfDone(tx, ch, *oldParent)
		}
		return nil
	})
}

// MoveToProject files the task in a project. A heading from another project
// is dropped, and a task filed in a project is no longer filed in an area.
func (s *TaskService) MoveToProject(id, projectID int64) error {
	return s.editTx(id, func(tx storage.Store, t *models.Task) error {
		if err := requireProject(tx, projectID); err != nil {
			return err
		}
		t.ProjectID, t.AreaID = &projectID, nil
		return s.dropForeignHeading(tx, t)
	})
}

// FileInArea files a top-level task directly in an area, outside any project.
func (s *TaskService) FileInArea(id, areaID int64) error {
	return s.editTx(id, func(tx storage.Store, t *models.Task) error {
		if _, err := getArea(tx, areaID); err != nil {
			return err
		}
		t.AreaID, t.ProjectID, t.HeadingID = &areaID, nil, nil
		return nil
	})
}

// Unfile removes the task from its area, project and heading.
func (s *TaskService) Unfile(id int64) error {
	return s.edit(id, func(t *models.Task) { t.AreaID, t.ProjectID, t.HeadingID = nil, nil, nil })
}

// SetHeading places the task under a heading (which also files it in that
// heading's project), or removes it from its heading when headingID is nil.
func (s *TaskService) SetHeading(id int64, headingID *int64) error {
	return s.editTx(id, func(tx storage.Store, t *models.Task) error {
		if headingID == nil {
			t.HeadingID = nil
			return nil
		}
		h, err := getHeading(tx, *headingID)
		if err != nil {
			return err
		}
		t.HeadingID, t.ProjectID, t.AreaID = headingID, &h.ProjectID, nil
		return nil
	})
}

// Plan sets the day the task is planned for.
func (s *TaskService) Plan(id int64, day time.Time) error {
	return s.edit(id, func(t *models.Task) { t.PlannedDate = &day })
}

// Delete removes the task together with its Subtasks.
func (s *TaskService) Delete(id int64) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		t, err := s.get(tx, id)
		if err != nil {
			return err
		}
		if err := s.deleteTree(tx, ch, id); err != nil {
			return err
		}
		if t.ParentID != nil {
			return s.completeParentIfDone(tx, ch, *t.ParentID)
		}
		return nil
	})
}

// Tag attaches a tag to the task, finding it by case-insensitive name or
// creating it. Attaching a tag the task already has changes nothing.
func (s *TaskService) Tag(id int64, name, color string) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		if _, err := s.get(tx, id); err != nil {
			return err
		}
		tag, err := findOrCreateTag(tx, name, color)
		if err != nil {
			return fmt.Errorf("could not create tag: %w", err)
		}
		if err := tx.AttachTagToTask(id, tag.ID); err != nil {
			return fmt.Errorf("could not attach tag: %w", err)
		}
		ch.add(id, ChangeTagged)
		return nil
	})
}

// Untag detaches the tag with the given (case-insensitive) name. The tag
// itself remains available for other tasks.
func (s *TaskService) Untag(id int64, name string) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		t, err := s.get(tx, id)
		if err != nil {
			return err
		}
		key := NormalizeTagName(name)
		for _, tag := range t.Tags {
			if NormalizeTagName(tag.Name) == key {
				if err := tx.DetachTagFromTask(id, tag.ID); err != nil {
					return err
				}
				ch.add(id, ChangeTagged)
				return nil
			}
		}
		return invalidError(fmt.Sprintf("task %d has no tag %q", id, strings.TrimSpace(name)))
	})
}

// StartTimer begins tracking time on the task. A timer already running on the
// same task is stopped first, so a task never has two open logs.
func (s *TaskService) StartTimer(id int64) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		if _, err := s.get(tx, id); err != nil {
			return err
		}
		if err := s.stopTimer(tx, id); err != nil {
			return err
		}
		if err := tx.OpenTimeLog(id, s.clock.Now()); err != nil {
			return err
		}
		ch.add(id, ChangeTimer)
		return s.record(tx, id, "started_timer", "Started tracking time")
	})
}

// StopTimer ends the task's running timer; it is a no-op when none runs.
func (s *TaskService) StopTimer(id int64) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		if _, err := s.get(tx, id); err != nil {
			return err
		}
		if err := s.stopTimer(tx, id); err != nil {
			return err
		}
		ch.add(id, ChangeTimer)
		return nil
	})
}

// History returns the task's events, newest first.
func (s *TaskService) History(id int64) ([]models.TaskHistory, error) {
	return s.store.GetTaskHistory(id)
}

func (s *TaskService) get(st storage.Store, id int64) (*models.Task, error) {
	t, err := st.GetTask(id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, &NotFoundError{Entity: "task", ID: id}
	}
	return t, err
}

func (s *TaskService) record(st storage.Store, taskID int64, action, details string) error {
	return st.AddTaskHistory(&models.TaskHistory{
		TaskID:    taskID,
		Action:    action,
		Details:   details,
		Timestamp: s.clock.Now(),
	})
}

func (s *TaskService) stopTimer(tx storage.Store, id int64) error {
	secs, closed, err := tx.CloseOpenTimeLog(id, s.clock.Now())
	if err != nil || !closed {
		return err
	}
	return s.record(tx, id, "stopped_timer",
		fmt.Sprintf("Stopped tracking time. Duration: %dh %dm %ds", secs/3600, (secs%3600)/60, secs%60))
}

// complete is the single completion path: a manual completion and the
// automatic completion of a parent both run through it.
func (s *TaskService) complete(tx storage.Store, ch *changes, id int64) error {
	t, err := s.get(tx, id)
	if err != nil {
		return err
	}
	if t.Status == models.StatusCompleted {
		return nil
	}
	kids, err := tx.GetSubtasks(id)
	if err != nil {
		return err
	}
	if open := countOpen(kids); open > 0 {
		return &OpenSubtasksError{TaskID: id, Count: open}
	}
	if t.IsRunning {
		if err := s.stopTimer(tx, id); err != nil {
			return err
		}
	}
	if err := s.save(tx, t, func(t *models.Task) { t.Status = models.StatusCompleted }); err != nil {
		return err
	}
	ch.add(id, ChangeStatus)
	if t.ParentID != nil {
		return s.completeParentIfDone(tx, ch, *t.ParentID)
	}
	return nil
}

// completeParentIfDone completes an open parent whose Subtasks are all closed
// and at least one of them completed.
func (s *TaskService) completeParentIfDone(tx storage.Store, ch *changes, parentID int64) error {
	parent, err := s.get(tx, parentID)
	if errors.Is(err, ErrNotFound) {
		return nil // removed in the same operation
	}
	if err != nil {
		return err
	}
	if parent.Status != models.StatusOpen {
		return nil
	}
	kids, err := tx.GetSubtasks(parentID)
	if err != nil {
		return err
	}
	completed := 0
	for _, k := range kids {
		switch k.Status {
		case models.StatusOpen:
			return nil
		case models.StatusCompleted:
			completed++
		}
	}
	if completed == 0 {
		return nil
	}
	return s.complete(tx, ch, parentID)
}

func (s *TaskService) reopen(tx storage.Store, ch *changes, id int64) error {
	t, err := s.get(tx, id)
	if err != nil {
		return err
	}
	if t.Status == models.StatusOpen {
		return nil
	}
	if err := s.save(tx, t, func(t *models.Task) { t.Status = models.StatusOpen }); err != nil {
		return err
	}
	ch.add(id, ChangeStatus)
	if t.ParentID != nil {
		parent, err := s.get(tx, *t.ParentID)
		if err != nil {
			return err
		}
		if parent.Status == models.StatusCompleted {
			return s.reopen(tx, ch, parent.ID)
		}
	}
	return nil
}

// cancelTree cancels t and every open descendant without touching ancestors.
func (s *TaskService) cancelTree(tx storage.Store, ch *changes, t *models.Task) error {
	kids, err := tx.GetSubtasks(t.ID)
	if err != nil {
		return err
	}
	for i := range kids {
		if kids[i].Status == models.StatusOpen {
			if err := s.cancelTree(tx, ch, &kids[i]); err != nil {
				return err
			}
		}
	}
	if t.IsRunning {
		if err := s.stopTimer(tx, t.ID); err != nil {
			return err
		}
	}
	if err := s.save(tx, t, func(t *models.Task) { t.Status = models.StatusCancelled }); err != nil {
		return err
	}
	ch.add(t.ID, ChangeStatus)
	return nil
}

func (s *TaskService) deleteTree(tx storage.Store, ch *changes, id int64) error {
	kids, err := tx.GetSubtasks(id)
	if err != nil {
		return err
	}
	for _, k := range kids {
		if err := s.deleteTree(tx, ch, k.ID); err != nil {
			return err
		}
	}
	if err := tx.DeleteTask(id); err != nil {
		return err
	}
	ch.add(id, ChangeDeleted)
	return nil
}

// checkAcyclic rejects nesting id under parentID when parentID is id or one of
// its descendants, by walking up from the new parent.
func (s *TaskService) checkAcyclic(tx storage.Store, id, parentID int64) error {
	seen := map[int64]bool{}
	for cur := &parentID; cur != nil; {
		if *cur == id {
			return &CycleError{TaskID: id, ParentID: parentID}
		}
		if seen[*cur] {
			return nil // pre-existing loop elsewhere; not caused by this change
		}
		seen[*cur] = true
		t, err := s.get(tx, *cur)
		if err != nil {
			return err
		}
		cur = t.ParentID
	}
	return nil
}

func (s *TaskService) edit(id int64, mutate func(*models.Task)) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		t, err := s.get(tx, id)
		if err != nil {
			return err
		}
		if err := s.save(tx, t, mutate); err != nil {
			return err
		}
		ch.add(id, ChangeUpdated)
		return nil
	})
}

// save applies mutate, persists, and records a status_changed or edited entry.
func (s *TaskService) save(tx storage.Store, t *models.Task, mutate func(*models.Task)) error {
	oldStatus := t.Status
	mutate(t)
	t.UpdatedAt = s.clock.Now()
	if err := tx.UpdateTask(t); err != nil {
		return err
	}
	if oldStatus != t.Status {
		return s.record(tx, t.ID, "status_changed",
			fmt.Sprintf("Status changed from '%s' to '%s'", oldStatus, t.Status))
	}
	return s.record(tx, t.ID, "edited", fmt.Sprintf("Task edited: '%s'", t.Title))
}

func countOpen(tasks []models.Task) int {
	n := 0
	for _, t := range tasks {
		if t.Status == models.StatusOpen {
			n++
		}
	}
	return n
}

func sameID(a, b *int64) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func requireProject(tx storage.Store, id int64) error {
	if _, err := tx.GetProject(id); errors.Is(err, storage.ErrNotFound) {
		return &NotFoundError{Entity: "project", ID: id}
	} else if err != nil {
		return err
	}
	return nil
}

// normalizeContainer checks a new task's container fields: every referenced
// area, project and heading exists, a heading implies its project, and the task
// is in a project or an area, never both.
func (s *TaskService) normalizeContainer(tx storage.Store, t *models.Task) error {
	if t.HeadingID != nil {
		h, err := getHeading(tx, *t.HeadingID)
		if err != nil {
			return err
		}
		if t.ProjectID == nil {
			t.ProjectID = &h.ProjectID
		} else if *t.ProjectID != h.ProjectID {
			return invalidError(fmt.Sprintf("heading %d belongs to project %d, not project %d", h.ID, h.ProjectID, *t.ProjectID))
		}
	}
	if t.ProjectID != nil && t.AreaID != nil {
		return invalidError("a task is filed in a project or an area, not both")
	}
	if t.ProjectID != nil {
		if err := requireProject(tx, *t.ProjectID); err != nil {
			return err
		}
	}
	if t.AreaID != nil {
		if _, err := getArea(tx, *t.AreaID); err != nil {
			return err
		}
	}
	return nil
}

// dropForeignHeading clears the task's heading when it belongs to a different project.
func (s *TaskService) dropForeignHeading(tx storage.Store, t *models.Task) error {
	if t.HeadingID == nil {
		return nil
	}
	h, err := tx.GetHeading(*t.HeadingID)
	if errors.Is(err, storage.ErrNotFound) || (err == nil && (t.ProjectID == nil || h.ProjectID != *t.ProjectID)) {
		t.HeadingID = nil
		return nil
	}
	return err
}

// editTx is edit with access to the transaction for validation.
func (s *TaskService) editTx(id int64, mutate func(tx storage.Store, t *models.Task) error) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		t, err := s.get(tx, id)
		if err != nil {
			return err
		}
		var mErr error
		if err := s.save(tx, t, func(t *models.Task) { mErr = mutate(tx, t) }); err != nil {
			return err
		}
		if mErr != nil {
			return mErr
		}
		ch.add(id, ChangeUpdated)
		return nil
	})
}

// deleteTasks removes tasks and their Subtasks. Tasks whose parent is not among
// them are the roots; deleting a root takes its subtree, so nothing is deleted
// twice. The parents left behind may complete, as with a single delete.
func (s *TaskService) deleteTasks(tx storage.Store, ch *changes, tasks []models.Task) error {
	in := map[int64]bool{}
	for _, t := range tasks {
		in[t.ID] = true
	}
	for _, t := range tasks {
		if t.ParentID != nil && in[*t.ParentID] {
			continue
		}
		if _, err := tx.GetTask(t.ID); errors.Is(err, storage.ErrNotFound) {
			continue // already removed as part of another subtree
		}
		if err := s.deleteTree(tx, ch, t.ID); err != nil {
			return err
		}
		if t.ParentID != nil {
			if err := s.completeParentIfDone(tx, ch, *t.ParentID); err != nil {
				return err
			}
		}
	}
	return nil
}
