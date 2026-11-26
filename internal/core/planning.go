package core

import (
	"context"
	"errors"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// ListPlanning is the list key for time blocks, carried by ListsInvalidated.
const ListPlanning = "planning"

const (
	minutesPerDay    = 24 * 60
	MinBlockMinutes  = 5
	maxEstimateHours = 24
)

// PlanningService owns Time Blocks: the slices of a day set aside for tasks.
// Blocks never overlap within a day, and never cross midnight.
type PlanningService struct {
	store storage.Store
	clock Clock
	bus   *bus
	lists *ListService

	busy    BusySource // optional: calendar busy times
	advisor Advisor    // optional: AI suggestions
}

// Interval is a stretch of one day, in minutes after midnight (end exclusive).
type Interval struct {
	Title    string
	StartMin int
	EndMin   int
}

// BusySource supplies busy times from outside Zenith (an ICS calendar feed).
// An error means the data may be stale; the intervals returned are still used.
type BusySource interface {
	Busy(ctx context.Context, date string) ([]Interval, error)
}

// SetBusySource wires calendar busy times into planning; nil removes them.
func (s *PlanningService) SetBusySource(b BusySource) { s.busy = b }

// DayPlan is everything the planner shows for one day.
type DayPlan struct {
	Date    string
	Tasks   []models.Task // the day's open tasks, in list order
	Blocks  []models.TimeBlock
	Busy    []Interval // calendar events; read-only
	BusyErr error      // why Busy may be missing or stale
}

// Plan returns the tasks to plan for a day and the blocks already placed on it.
// For today that is the Today list; for any other day it is the open tasks
// scheduled exactly for that day.
func (s *PlanningService) Plan(ctx context.Context, date string) (*DayPlan, error) {
	if err := checkDate(date); err != nil {
		return nil, err
	}
	var tasks []models.Task
	if date == s.lists.ToDay() {
		t, err := s.lists.Today()
		if err != nil {
			return nil, err
		}
		tasks = append(append(tasks, t.Day...), t.Evening...)
	} else {
		all, err := s.store.GetTasks()
		if err != nil {
			return nil, err
		}
		for _, t := range all {
			if isOpenTop(t) && t.When.Kind == models.WhenScheduled && t.When.Date == date {
				tasks = append(tasks, t)
			}
		}
	}
	blocks, err := s.store.GetTimeBlocks(date)
	if err != nil {
		return nil, err
	}
	plan := &DayPlan{Date: date, Tasks: tasks, Blocks: blocks}
	if s.busy != nil {
		plan.Busy, plan.BusyErr = s.busy.Busy(ctx, date)
	}
	return plan, nil
}

// AddBlock places a block for an open task. The task is not rescheduled: a
// block is where the user will work, When stays what they intended.
func (s *PlanningService) AddBlock(taskID int64, date string, startMin, durationMin int) (*models.TimeBlock, error) {
	if err := checkSlot(date, startMin, durationMin); err != nil {
		return nil, err
	}
	b := &models.TimeBlock{TaskID: taskID, Date: date, StartMin: startMin, DurationMin: durationMin}
	err := s.store.WithTx(func(tx storage.Store) error {
		t, err := tx.GetTask(taskID)
		if errors.Is(err, storage.ErrNotFound) {
			return &NotFoundError{Entity: "task", ID: taskID}
		}
		if err != nil {
			return err
		}
		if t.Status != models.StatusOpen {
			return invalidError("only open tasks can be time-blocked")
		}
		if err := checkFree(tx, b); err != nil {
			return err
		}
		return tx.AddTimeBlock(b)
	})
	if err != nil {
		return nil, err
	}
	s.changed()
	return b, nil
}

// Move puts a block at a new day and start; its duration is kept.
func (s *PlanningService) Move(id int64, date string, startMin int) error {
	return s.edit(id, func(b *models.TimeBlock) { b.Date, b.StartMin = date, startMin })
}

// Resize changes a block's duration; its start is kept.
func (s *PlanningService) Resize(id int64, durationMin int) error {
	return s.edit(id, func(b *models.TimeBlock) { b.DurationMin = durationMin })
}

// Remove deletes a block. Removing one that is already gone does nothing.
func (s *PlanningService) Remove(id int64) error {
	if err := s.store.DeleteTimeBlock(id); err != nil {
		return err
	}
	s.changed()
	return nil
}

func (s *PlanningService) edit(id int64, mutate func(*models.TimeBlock)) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		b, err := tx.GetTimeBlock(id)
		if errors.Is(err, storage.ErrNotFound) {
			return &NotFoundError{Entity: "time block", ID: id}
		}
		if err != nil {
			return err
		}
		mutate(b)
		if err := checkSlot(b.Date, b.StartMin, b.DurationMin); err != nil {
			return err
		}
		if err := checkFree(tx, b); err != nil {
			return err
		}
		b.UpdatedAt = s.clock.Now()
		return tx.UpdateTimeBlock(b)
	})
	if err != nil {
		return err
	}
	s.changed()
	return nil
}

func (s *PlanningService) changed() {
	s.bus.publish(ListsInvalidated{Keys: []string{ListPlanning}})
}

func checkDate(date string) error {
	if _, err := time.Parse(models.DateLayout, date); err != nil {
		return invalidError("invalid day " + date + " (want YYYY-MM-DD)")
	}
	return nil
}

func checkSlot(date string, startMin, durationMin int) error {
	if err := checkDate(date); err != nil {
		return err
	}
	if startMin < 0 || startMin >= minutesPerDay {
		return invalidError("a time block must start between 00:00 and 23:59")
	}
	if durationMin < MinBlockMinutes {
		return invalidError("a time block must be at least 5 minutes long")
	}
	if startMin+durationMin > minutesPerDay {
		return invalidError("a time block cannot run past midnight")
	}
	return nil
}

// checkFree refuses a block that overlaps another on its day.
func checkFree(tx storage.Store, b *models.TimeBlock) error {
	others, err := tx.GetTimeBlocks(b.Date)
	if err != nil {
		return err
	}
	for _, o := range others {
		if o.ID != b.ID && b.StartMin < o.EndMin() && o.StartMin < b.EndMin() {
			return invalidError("that time is already blocked")
		}
	}
	return nil
}

// SetEstimate sets a task's planned effort in minutes; 0 clears it.
func (s *TaskService) SetEstimate(id int64, minutes int) error {
	if minutes < 0 || minutes > maxEstimateHours*60 {
		return invalidError("an estimate must be between 0 and 24 hours")
	}
	return s.edit(id, func(t *models.Task) { t.EstimateMinutes = minutes })
}
