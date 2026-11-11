package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// Task statuses.
const (
	StatusTodo = "todo"
	StatusDone = "done"
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
	ProjectID *int64
	DueDate   *time.Time
	Priority  models.Priority
	Recurring string
}

func (s *TaskService) Add(in AddTaskInput) (*models.Task, error) {
	task := &models.Task{
		Title:     in.Title,
		Status:    StatusTodo,
		Priority:  in.Priority,
		ProjectID: in.ProjectID,
		DueDate:   in.DueDate,
		Recurring: in.Recurring,
	}
	err := s.store.WithTx(func(tx storage.Store) error {
		if err := tx.AddTask(task); err != nil {
			return err
		}
		return s.record(tx, task.ID, "created", fmt.Sprintf("Task created: '%s'", task.Title))
	})
	if err != nil {
		return nil, err
	}
	s.changed(task.ID, ChangeCreated)
	return task, nil
}

func (s *TaskService) List() ([]models.Task, error) {
	return s.store.GetTasks()
}

// Get returns the task or a *NotFoundError.
func (s *TaskService) Get(id int64) (*models.Task, error) {
	return s.get(s.store, id)
}

// Complete marks the task done, stopping its running timer first.
func (s *TaskService) Complete(id int64) error {
	return s.setStatus(id, StatusDone)
}

// Reopen puts a finished task back to todo.
func (s *TaskService) Reopen(id int64) error {
	return s.setStatus(id, StatusTodo)
}

// MoveToProject assigns the task to a project.
func (s *TaskService) MoveToProject(id, projectID int64) error {
	return s.edit(id, func(t *models.Task) { t.ProjectID = &projectID })
}

// Plan sets the day the task is planned for.
func (s *TaskService) Plan(id int64, day time.Time) error {
	return s.edit(id, func(t *models.Task) { t.PlannedDate = &day })
}

func (s *TaskService) Delete(id int64) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		if _, err := s.get(tx, id); err != nil {
			return err
		}
		return tx.DeleteTask(id)
	})
	if err == nil {
		s.changed(id, ChangeDeleted)
	}
	return err
}

// Tag attaches a tag, creating it by name when needed.
func (s *TaskService) Tag(id int64, name, color string) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		if _, err := s.get(tx, id); err != nil {
			return err
		}
		tag := &models.Tag{Name: name, Color: color}
		if err := tx.AddTag(tag); err != nil {
			return fmt.Errorf("could not create tag: %w", err)
		}
		if err := tx.AttachTagToTask(id, tag.ID); err != nil {
			return fmt.Errorf("could not attach tag: %w", err)
		}
		return nil
	})
	if err == nil {
		s.changed(id, ChangeTagged)
	}
	return err
}

// StartTimer begins tracking time on the task. A timer already running on the
// same task is stopped first, so a task never has two open logs.
func (s *TaskService) StartTimer(id int64) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		if _, err := s.get(tx, id); err != nil {
			return err
		}
		if err := s.stopTimer(tx, id); err != nil {
			return err
		}
		if err := tx.OpenTimeLog(id, s.clock.Now()); err != nil {
			return err
		}
		return s.record(tx, id, "started_timer", "Started tracking time")
	})
	if err == nil {
		s.changed(id, ChangeTimer)
	}
	return err
}

// StopTimer ends the task's running timer; it is a no-op when none runs.
func (s *TaskService) StopTimer(id int64) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		if _, err := s.get(tx, id); err != nil {
			return err
		}
		return s.stopTimer(tx, id)
	})
	if err == nil {
		s.changed(id, ChangeTimer)
	}
	return err
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

func (s *TaskService) setStatus(id int64, status string) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		t, err := s.get(tx, id)
		if err != nil {
			return err
		}
		if status == StatusDone && t.IsRunning {
			if err := s.stopTimer(tx, id); err != nil {
				return err
			}
		}
		return s.save(tx, t, func(t *models.Task) { t.Status = status })
	})
	if err == nil {
		s.changed(id, ChangeStatus)
	}
	return err
}

func (s *TaskService) edit(id int64, mutate func(*models.Task)) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		t, err := s.get(tx, id)
		if err != nil {
			return err
		}
		return s.save(tx, t, mutate)
	})
	if err == nil {
		s.changed(id, ChangeUpdated)
	}
	return err
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

// changed announces a committed task mutation.
func (s *TaskService) changed(id int64, kind ChangeKind) {
	s.bus.publish(TaskChanged{TaskID: id, Kind: kind}, ListsInvalidated{Keys: []string{ListTasks}})
}
