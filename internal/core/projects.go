package core

import (
	"errors"
	"strings"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// ProjectService owns Project rules: a Project is a finite outcome with its own
// completion state, inside an Area or standalone, holding Headings and Tasks.
type ProjectService struct {
	store storage.Store
	clock Clock
	bus   *bus
	tasks *TaskService
}

func requireName(kind, name string) (string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", invalidError(kind + " name cannot be empty")
	}
	return name, nil
}

func (s *ProjectService) get(st storage.Store, id int64) (*models.Project, error) {
	p, err := st.GetProject(id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, &NotFoundError{Entity: "project", ID: id}
	}
	return p, err
}

func (s *ProjectService) announce() {
	s.bus.publish(ListsInvalidated{Keys: []string{ListProjects}})
}

// Add creates an open Project, standalone when areaID is nil.
func (s *ProjectService) Add(name string, areaID *int64) (*models.Project, error) {
	name, err := requireName("project", name)
	if err != nil {
		return nil, err
	}
	p := &models.Project{Name: name, AreaID: areaID, Status: models.StatusOpen}
	err = s.store.WithTx(func(tx storage.Store) error {
		if areaID != nil {
			if _, err := getArea(tx, *areaID); err != nil {
				return err
			}
		}
		return tx.AddProject(p)
	})
	if err != nil {
		return nil, err
	}
	s.announce()
	return p, nil
}

func (s *ProjectService) Get(id int64) (*models.Project, error) { return s.get(s.store, id) }

func (s *ProjectService) List() ([]models.Project, error) { return s.store.GetProjects() }

func (s *ProjectService) update(id int64, mutate func(tx storage.Store, p *models.Project) error) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		p, err := s.get(tx, id)
		if err != nil {
			return err
		}
		if err := mutate(tx, p); err != nil {
			return err
		}
		p.UpdatedAt = s.clock.Now()
		return tx.UpdateProject(p)
	})
	if err == nil {
		s.announce()
	}
	return err
}

func (s *ProjectService) Rename(id int64, name string) error {
	name, err := requireName("project", name)
	if err != nil {
		return err
	}
	return s.update(id, func(_ storage.Store, p *models.Project) error { p.Name = name; return nil })
}

// Move files the project under an Area, or makes it standalone when areaID is nil.
func (s *ProjectService) Move(id int64, areaID *int64) error {
	return s.update(id, func(tx storage.Store, p *models.Project) error {
		if areaID != nil {
			if _, err := getArea(tx, *areaID); err != nil {
				return err
			}
		}
		p.AreaID = areaID
		return nil
	})
}

// Complete finishes the project. It is refused while the project has open
// tasks, mirroring the rule for a Task with open Subtasks.
func (s *ProjectService) Complete(id int64) error {
	return s.update(id, func(tx storage.Store, p *models.Project) error {
		if p.Status == models.StatusCompleted {
			return nil
		}
		tasks, err := tx.GetTasksInProject(id)
		if err != nil {
			return err
		}
		if n := countOpen(tasks); n > 0 {
			return &OpenTasksError{Entity: "project", ID: id, Count: n}
		}
		p.Status = models.StatusCompleted
		return nil
	})
}

// Cancel cancels the project and every open task in it.
func (s *ProjectService) Cancel(id int64) error {
	err := s.tasks.run(func(tx storage.Store, ch *changes) error {
		p, err := s.get(tx, id)
		if err != nil {
			return err
		}
		if p.Status == models.StatusCancelled {
			return nil
		}
		tasks, err := tx.GetTasksInProject(id)
		if err != nil {
			return err
		}
		for _, t := range tasks {
			// Re-read: an earlier task's cancel may already have closed this one as a subtask.
			cur, err := tx.GetTask(t.ID)
			if err != nil {
				return err
			}
			if cur.Status != models.StatusOpen {
				continue
			}
			if err := s.tasks.cancel(tx, ch, t.ID); err != nil {
				return err
			}
		}
		p.Status = models.StatusCancelled
		p.UpdatedAt = s.clock.Now()
		return tx.UpdateProject(p)
	})
	if err == nil {
		s.announce()
	}
	return err
}

// Reopen makes a completed or cancelled project open again; its tasks are left as they are.
func (s *ProjectService) Reopen(id int64) error {
	return s.update(id, func(_ storage.Store, p *models.Project) error { p.Status = models.StatusOpen; return nil })
}

// Delete removes the project with its Headings and every Task in it (and their
// Subtasks), leaving nothing behind.
func (s *ProjectService) Delete(id int64) error {
	err := s.tasks.run(func(tx storage.Store, ch *changes) error {
		if _, err := s.get(tx, id); err != nil {
			return err
		}
		tasks, err := tx.GetTasksInProject(id)
		if err != nil {
			return err
		}
		if err := s.tasks.deleteTasks(tx, ch, tasks); err != nil {
			return err
		}
		headings, err := tx.GetHeadings(id)
		if err != nil {
			return err
		}
		for _, h := range headings {
			if err := tx.DeleteHeading(h.ID); err != nil {
				return err
			}
		}
		return tx.DeleteProject(id)
	})
	if err == nil {
		s.announce()
		s.bus.publish(ListsInvalidated{Keys: []string{ListHeadings}})
	}
	return err
}
