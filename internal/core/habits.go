package core

import (
	"errors"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// DefaultHabitFrequency is used for habits created without one.
const DefaultHabitFrequency = "daily"

// HabitService owns habit rules.
type HabitService struct {
	store storage.Store
	bus   *bus
}

func (s *HabitService) Add(name string) (*models.Habit, error) {
	h := &models.Habit{Name: name, Frequency: DefaultHabitFrequency}
	if err := s.store.AddHabit(h); err != nil {
		return nil, err
	}
	s.bus.publish(ListsInvalidated{Keys: []string{ListHabits}})
	return h, nil
}

func (s *HabitService) List() ([]models.Habit, error) {
	return s.store.GetHabits()
}

// Log records a completion of the habit now.
func (s *HabitService) Log(id int64) error {
	if _, err := s.store.GetHabit(id); errors.Is(err, storage.ErrNotFound) {
		return &NotFoundError{Entity: "habit", ID: id}
	} else if err != nil {
		return err
	}
	if err := s.store.LogHabit(id); err != nil {
		return err
	}
	s.bus.publish(ListsInvalidated{Keys: []string{ListHabits}})
	return nil
}
