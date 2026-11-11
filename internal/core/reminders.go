package core

import (
	"errors"
	"fmt"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// ReminderService schedules reminders and delivers the ones that are due.
//
// Delivery is at-most-once across processes: Tick claims each due reminder
// with one guarded UPDATE (ADR 0002) before notifying, so the CLI's passive
// check, `zenith daemon` and the desktop app can all tick without ever firing
// the same reminder twice. If the notification itself fails the reminder stays
// claimed and the failure is returned; it is not retried.
type ReminderService struct {
	store    storage.Store
	clock    Clock
	notifier Notifier
	bus      *bus
}

// Add schedules a reminder to fire after d. taskID is optional.
func (s *ReminderService) Add(title string, d time.Duration, taskID *int64) (*models.ScheduledTimer, error) {
	if taskID != nil {
		if _, err := s.store.GetTask(*taskID); errors.Is(err, storage.ErrNotFound) {
			return nil, &NotFoundError{Entity: "task", ID: *taskID}
		} else if err != nil {
			return nil, err
		}
	}
	r := &models.ScheduledTimer{
		Title:       title,
		TaskID:      taskID,
		TriggerTime: s.clock.Now().Add(d),
		Status:      "pending",
	}
	if err := s.store.AddScheduledTimer(r); err != nil {
		return nil, err
	}
	return r, nil
}

// Pending lists reminders that have not fired yet.
func (s *ReminderService) Pending() ([]models.ScheduledTimer, error) {
	return s.store.GetPendingScheduledTimers()
}

// Tick delivers every reminder due at now and returns the ones this call
// claimed. Delivery errors (for example notify.ErrUnavailable) are joined into
// the returned error; the returned reminders are still valid.
func (c *Core) Tick(now time.Time) ([]models.ScheduledTimer, error) {
	return c.Reminders.tick(now)
}

func (s *ReminderService) tick(now time.Time) ([]models.ScheduledTimer, error) {
	pending, err := s.store.GetPendingScheduledTimers()
	if err != nil {
		return nil, err
	}
	var fired []models.ScheduledTimer
	var errs []error
	for _, r := range pending {
		if r.TriggerTime.After(now) {
			continue
		}
		won, err := s.store.ClaimScheduledTimer(r.ID)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if !won {
			continue // another process fired it
		}
		fired = append(fired, r)
		if err := s.notifier.Notify("Zenith Reminder", r.Title); err != nil {
			errs = append(errs, fmt.Errorf("reminder %d: %w", r.ID, err))
		}
		s.bus.publish(ReminderDue{ReminderID: r.ID})
	}
	return fired, errors.Join(errs...)
}
