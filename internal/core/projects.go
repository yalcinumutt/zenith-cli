package core

import (
	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// ProjectService owns project rules.
type ProjectService struct {
	store storage.Store
	bus   *bus
}

func (s *ProjectService) Add(name string) (*models.Project, error) {
	p := &models.Project{Name: name}
	if err := s.store.AddProject(p); err != nil {
		return nil, err
	}
	s.bus.publish(ListsInvalidated{Keys: []string{ListProjects}})
	return p, nil
}

func (s *ProjectService) List() ([]models.Project, error) {
	return s.store.GetProjects()
}
