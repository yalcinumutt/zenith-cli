package core

import (
	"errors"
	"sort"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// DefaultHabitFrequency is used for habits created without one.
const DefaultHabitFrequency = "daily"

// HabitService owns habit rules.
type HabitService struct {
	store storage.Store
	bus   *bus
	clock Clock
}

func (s *HabitService) Add(name string) (*models.Habit, error) {
	name, err := requireName("habit", name)
	if err != nil {
		return nil, err
	}
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

func (s *HabitService) get(id int64) (*models.Habit, error) {
	h, err := s.store.GetHabit(id)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, &NotFoundError{Entity: "habit", ID: id}
	}
	return h, err
}

// Rename changes a habit's name.
func (s *HabitService) Rename(id int64, name string) error {
	name, err := requireName("habit", name)
	if err != nil {
		return err
	}
	h, err := s.get(id)
	if err != nil {
		return err
	}
	h.Name, h.UpdatedAt = name, s.clock.Now()
	if err := s.store.UpdateHabit(h); err != nil {
		return err
	}
	s.bus.publish(ListsInvalidated{Keys: []string{ListHabits}})
	return nil
}

// Delete removes a habit and, with it, its history from view.
func (s *HabitService) Delete(id int64) error {
	if _, err := s.get(id); err != nil {
		return err
	}
	if err := s.store.DeleteHabit(id); err != nil {
		return err
	}
	s.bus.publish(ListsInvalidated{Keys: []string{ListHabits}})
	return nil
}

// dayStart is local midnight of a YYYY-MM-DD day in the clock's zone.
func (s *HabitService) dayStart(day string) (time.Time, error) {
	t, err := time.ParseInLocation(models.DateLayout, day, s.clock.Now().Location())
	if err != nil {
		return time.Time{}, invalidError("invalid day " + day + " (want YYYY-MM-DD)")
	}
	return t, nil
}

// SetDay marks a habit done or not done on one calendar day. Marking done is
// idempotent (one completion per habit per day is enough); marking not done
// removes every completion on that day. Future days are refused.
func (s *HabitService) SetDay(id int64, day string, done bool) error {
	if _, err := s.get(id); err != nil {
		return err
	}
	start, err := s.dayStart(day)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	end := start.AddDate(0, 0, 1)
	if start.After(now) {
		return invalidError("cannot log a habit in the future")
	}
	err = s.store.WithTx(func(tx storage.Store) error {
		logs, err := tx.GetHabitLogs(start, end)
		if err != nil {
			return err
		}
		var mine []models.HabitLog
		for _, l := range logs {
			if l.HabitID == id {
				mine = append(mine, l)
			}
		}
		switch {
		case done && len(mine) == 0:
			at := start.Add(12 * time.Hour) // a past day: midday, so no zone shifts it onto another date
			if now.Before(end) {
				at = now
			}
			return tx.LogHabitAt(id, at)
		case !done:
			for _, l := range mine {
				if err := tx.DeleteHabitLog(l.ID); err != nil {
					return err
				}
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.bus.publish(ListsInvalidated{Keys: []string{ListHabits}})
	return nil
}

// Days returns, for every habit with at least one completion in the last n
// days (today included), the distinct local days it was done, oldest first.
func (s *HabitService) Days(n int) (map[int64][]string, error) {
	now := s.clock.Now()
	today, err := s.dayStart(now.Format(models.DateLayout))
	if err != nil {
		return nil, err
	}
	logs, err := s.store.GetHabitLogs(today.AddDate(0, 0, -(n-1)), today.AddDate(0, 0, 1))
	if err != nil {
		return nil, err
	}
	seen := map[int64]map[string]bool{}
	for _, l := range logs {
		d := l.CompletedAt.In(now.Location()).Format(models.DateLayout)
		if seen[l.HabitID] == nil {
			seen[l.HabitID] = map[string]bool{}
		}
		seen[l.HabitID][d] = true
	}
	out := make(map[int64][]string, len(seen))
	for id, set := range seen {
		for d := range set {
			out[id] = append(out[id], d)
		}
		sort.Strings(out[id])
	}
	return out, nil
}
