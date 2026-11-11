package core

import (
	"errors"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

func getArea(st storage.Store, id int64) (*models.Area, error) {
	a, err := st.GetArea(id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, &NotFoundError{Entity: "area", ID: id}
	}
	return a, err
}

// AreaService owns Area rules. An Area is a long-lived bucket of
// responsibility: it holds Projects and top-level Tasks and never completes.
type AreaService struct {
	store storage.Store
	clock Clock
	bus   *bus
	tasks *TaskService
}

func (s *AreaService) announce() { s.bus.publish(ListsInvalidated{Keys: []string{ListAreas}}) }

func (s *AreaService) Add(name string) (*models.Area, error) {
	name, err := requireName("area", name)
	if err != nil {
		return nil, err
	}
	a := &models.Area{Name: name}
	if err := s.store.AddArea(a); err != nil {
		return nil, err
	}
	s.announce()
	return a, nil
}

func (s *AreaService) Get(id int64) (*models.Area, error) { return getArea(s.store, id) }

func (s *AreaService) List() ([]models.Area, error) { return s.store.GetAreas() }

func (s *AreaService) Rename(id int64, name string) error {
	name, err := requireName("area", name)
	if err != nil {
		return err
	}
	err = s.store.WithTx(func(tx storage.Store) error {
		a, err := getArea(tx, id)
		if err != nil {
			return err
		}
		a.Name = name
		a.UpdatedAt = s.clock.Now()
		return tx.UpdateArea(a)
	})
	if err == nil {
		s.announce()
	}
	return err
}

// Delete removes the Area but keeps what was in it: its Projects become
// standalone and its top-level Tasks become unfiled. Nothing is deleted with it.
func (s *AreaService) Delete(id int64) error {
	err := s.tasks.run(func(tx storage.Store, ch *changes) error {
		if _, err := getArea(tx, id); err != nil {
			return err
		}
		projects, err := tx.GetProjects()
		if err != nil {
			return err
		}
		for _, p := range projects {
			if p.AreaID != nil && *p.AreaID == id {
				p.AreaID = nil
				p.UpdatedAt = s.clock.Now()
				if err := tx.UpdateProject(&p); err != nil {
					return err
				}
			}
		}
		tasks, err := tx.GetTasksInArea(id)
		if err != nil {
			return err
		}
		for i := range tasks {
			if err := s.tasks.save(tx, &tasks[i], func(t *models.Task) { t.AreaID = nil }); err != nil {
				return err
			}
			ch.add(tasks[i].ID, ChangeUpdated)
		}
		return tx.DeleteArea(id)
	})
	if err == nil {
		s.announce()
		s.bus.publish(ListsInvalidated{Keys: []string{ListProjects}})
	}
	return err
}

// HeadingService owns Heading rules. A Heading groups Tasks inside one Project;
// it can hold nothing but Tasks and has no completion state.
type HeadingService struct {
	store storage.Store
	clock Clock
	bus   *bus
	tasks *TaskService
}

func getHeading(st storage.Store, id int64) (*models.Heading, error) {
	h, err := st.GetHeading(id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, &NotFoundError{Entity: "heading", ID: id}
	}
	return h, err
}

func (s *HeadingService) announce() { s.bus.publish(ListsInvalidated{Keys: []string{ListHeadings}}) }

func (s *HeadingService) Add(projectID int64, title string) (*models.Heading, error) {
	title, err := requireName("heading", title)
	if err != nil {
		return nil, err
	}
	h := &models.Heading{ProjectID: projectID, Title: title}
	err = s.store.WithTx(func(tx storage.Store) error {
		if _, err := tx.GetProject(projectID); errors.Is(err, storage.ErrNotFound) {
			return &NotFoundError{Entity: "project", ID: projectID}
		} else if err != nil {
			return err
		}
		return tx.AddHeading(h)
	})
	if err != nil {
		return nil, err
	}
	s.announce()
	return h, nil
}

func (s *HeadingService) Get(id int64) (*models.Heading, error) { return getHeading(s.store, id) }

// List returns the headings of one project, oldest first.
func (s *HeadingService) List(projectID int64) ([]models.Heading, error) {
	if _, err := s.store.GetProject(projectID); errors.Is(err, storage.ErrNotFound) {
		return nil, &NotFoundError{Entity: "project", ID: projectID}
	} else if err != nil {
		return nil, err
	}
	return s.store.GetHeadings(projectID)
}

func (s *HeadingService) Rename(id int64, title string) error {
	title, err := requireName("heading", title)
	if err != nil {
		return err
	}
	err = s.store.WithTx(func(tx storage.Store) error {
		h, err := getHeading(tx, id)
		if err != nil {
			return err
		}
		h.Title = title
		h.UpdatedAt = s.clock.Now()
		return tx.UpdateHeading(h)
	})
	if err == nil {
		s.announce()
	}
	return err
}

// Move relocates the heading to another project, taking its tasks along.
func (s *HeadingService) Move(id, projectID int64) error {
	err := s.tasks.run(func(tx storage.Store, ch *changes) error {
		h, err := getHeading(tx, id)
		if err != nil {
			return err
		}
		if h.ProjectID == projectID {
			return nil
		}
		if _, err := tx.GetProject(projectID); errors.Is(err, storage.ErrNotFound) {
			return &NotFoundError{Entity: "project", ID: projectID}
		} else if err != nil {
			return err
		}
		tasks, err := tx.GetTasksInHeading(id)
		if err != nil {
			return err
		}
		for i := range tasks {
			if err := s.tasks.save(tx, &tasks[i], func(t *models.Task) { t.ProjectID = &projectID; t.AreaID = nil }); err != nil {
				return err
			}
			ch.add(tasks[i].ID, ChangeUpdated)
		}
		h.ProjectID = projectID
		h.UpdatedAt = s.clock.Now()
		return tx.UpdateHeading(h)
	})
	if err == nil {
		s.announce()
	}
	return err
}

// Delete removes the heading together with the tasks under it.
func (s *HeadingService) Delete(id int64) error {
	err := s.tasks.run(func(tx storage.Store, ch *changes) error {
		if _, err := getHeading(tx, id); err != nil {
			return err
		}
		tasks, err := tx.GetTasksInHeading(id)
		if err != nil {
			return err
		}
		if err := s.tasks.deleteTasks(tx, ch, tasks); err != nil {
			return err
		}
		return tx.DeleteHeading(id)
	})
	if err == nil {
		s.announce()
	}
	return err
}
