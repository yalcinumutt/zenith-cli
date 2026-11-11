package core

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// Repeating tasks (ADR 0006). A Repeater holds the rule and a template; each
// occurrence is an ordinary Task pointing back at it. Completing an occurrence is
// a normal completion, and the core then generates the next one. The completed
// occurrence stays in the Logbook. The template follows the most recently
// completed occurrence, so edits to an occurrence carry forward.

func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func midnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// NextOccurrence returns the first day the rule produces strictly after from.
// Weeks start on Monday for "every N weeks on ..." rules.
func NextOccurrence(r models.RepeatRule, from time.Time) time.Time {
	from = midnight(from)
	n := r.Interval
	if n < 1 {
		n = 1
	}
	switch r.Kind {
	case models.RepeatWeeks:
		if len(r.Weekdays) == 0 {
			return from.AddDate(0, 0, 7*n)
		}
		idx := func(d int) int { return (d + 6) % 7 } // Monday = 0
		cur := idx(int(from.Weekday()))
		var set []int
		for _, d := range r.Weekdays {
			set = append(set, idx(d))
		}
		sort.Ints(set)
		for _, w := range set {
			if w > cur {
				return from.AddDate(0, 0, w-cur)
			}
		}
		// Nothing later this week: the first listed day, n weeks on from this week's Monday.
		return from.AddDate(0, 0, (7-cur)+set[0]+7*(n-1))
	case models.RepeatMonths:
		day := r.MonthDay
		if day < 1 {
			day = from.Day()
		}
		first := time.Date(from.Year(), from.Month()+time.Month(n), 1, 0, 0, 0, 0, time.UTC)
		if dim := daysIn(first.Year(), first.Month()); day > dim {
			day = dim
		}
		return time.Date(first.Year(), first.Month(), day, 0, 0, 0, 0, time.UTC)
	}
	return from.AddDate(0, 0, n)
}

// normalizeRule validates a rule and fills defaults. anchor is the occurrence's
// day, used when a monthly rule names no day.
func normalizeRule(r models.RepeatRule, anchor time.Time) (models.RepeatRule, error) {
	if r.Interval == 0 {
		r.Interval = 1
	}
	if r.Interval < 1 || r.Interval > 1000 {
		return r, invalidError("repeat interval must be between 1 and 1000")
	}
	switch r.Kind {
	case models.RepeatDays:
		r.Weekdays, r.MonthDay = nil, 0
	case models.RepeatWeeks:
		r.MonthDay = 0
		seen := map[int]bool{}
		var days []int
		for _, d := range r.Weekdays {
			if d < 0 || d > 6 {
				return r, invalidError(fmt.Sprintf("invalid weekday %d (0=Sunday .. 6=Saturday)", d))
			}
			if !seen[d] {
				seen[d] = true
				days = append(days, d)
			}
		}
		sort.Ints(days)
		r.Weekdays = days
	case models.RepeatMonths:
		r.Weekdays = nil
		if r.MonthDay == 0 {
			r.MonthDay = anchor.Day()
		}
		if r.MonthDay < 1 || r.MonthDay > 31 {
			return r, invalidError("day of month must be between 1 and 31")
		}
	default:
		return r, invalidError(fmt.Sprintf("unknown repeat kind %q (use days, weeks or months)", r.Kind))
	}
	return r, nil
}

// ParseWeekdays reads "mon,wed,fri" (names or 0-6) into weekday numbers.
func ParseWeekdays(s string) ([]int, error) {
	names := map[string]int{"sun": 0, "mon": 1, "tue": 2, "wed": 3, "thu": 4, "fri": 5, "sat": 6}
	var out []int
	for _, p := range strings.Split(s, ",") {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		if len(p) >= 3 {
			if d, ok := names[p[:3]]; ok {
				out = append(out, d)
				continue
			}
		}
		var d int
		if _, err := fmt.Sscanf(p, "%d", &d); err == nil && d >= 0 && d <= 6 {
			out = append(out, d)
			continue
		}
		return nil, invalidError(fmt.Sprintf("unknown weekday %q", p))
	}
	return out, nil
}

// occurrenceDay is the day an occurrence belongs to: its generated day, else its
// scheduled day, else today.
func (s *TaskService) occurrenceDay(t *models.Task) time.Time {
	for _, d := range []string{t.OccurrenceDate, t.When.Date} {
		if d != "" {
			if day, err := time.Parse(models.DateLayout, d); err == nil {
				return day
			}
		}
	}
	return midnight(s.clock.Now())
}

func deadlineOffset(deadline string, day time.Time) *int {
	if deadline == "" {
		return nil
	}
	d, err := time.Parse(models.DateLayout, deadline)
	if err != nil {
		return nil
	}
	off := int(d.Sub(day).Hours() / 24)
	return &off
}

// attachRepeater creates a Repeater from the task and makes the task its current
// occurrence (scheduling it on its occurrence day if it had no date). The task
// row is not saved; the caller does that. A task that already repeats just gets
// its rule replaced.
func (s *TaskService) attachRepeater(tx storage.Store, t *models.Task, rule models.RepeatRule) error {
	day := s.occurrenceDay(t)
	rule, err := normalizeRule(rule, day)
	if err != nil {
		return err
	}
	if t.RepeaterID != nil {
		if r, err := tx.GetRepeater(*t.RepeaterID); err == nil {
			r.Rule, r.UpdatedAt = rule, s.clock.Now()
			return tx.UpdateRepeater(r)
		} else if !errors.Is(err, storage.ErrNotFound) {
			return err
		}
	}
	r := &models.Repeater{
		Rule: rule, Title: t.Title, Description: t.Description,
		ProjectID: t.ProjectID, AreaID: t.AreaID, HeadingID: t.HeadingID,
		Priority: t.Priority, Evening: t.Evening, DeadlineOffset: deadlineOffset(t.Deadline, day),
	}
	if err := tx.AddRepeater(r); err != nil {
		return err
	}
	t.RepeaterID, t.OccurrenceDate = &r.ID, day.Format(models.DateLayout)
	if t.When.Kind != models.WhenScheduled {
		t.When = models.When{Kind: models.WhenScheduled, Date: t.OccurrenceDate}
	}
	return nil
}

// generateNext creates the occurrence after a completed one. It is idempotent:
// if the next day already has an occurrence (for example after a reopen and a
// second completion), nothing more is created.
func (s *TaskService) generateNext(tx storage.Store, ch *changes, done *models.Task) error {
	if done.RepeaterID == nil {
		return nil
	}
	r, err := tx.GetRepeater(*done.RepeaterID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil // the series was removed
	}
	if err != nil {
		return err
	}
	day := s.occurrenceDay(done)
	next := NextOccurrence(r.Rule, day)
	nextStr := next.Format(models.DateLayout)
	if _, err := tx.FindOccurrence(r.ID, nextStr); err == nil {
		return nil
	} else if !errors.Is(err, storage.ErrNotFound) {
		return err
	}

	// The template follows the occurrence just completed.
	r.Title, r.Description = done.Title, done.Description
	r.ProjectID, r.AreaID, r.HeadingID = done.ProjectID, done.AreaID, done.HeadingID
	r.Priority, r.Evening = done.Priority, done.Evening
	r.DeadlineOffset = deadlineOffset(done.Deadline, day)
	r.UpdatedAt = s.clock.Now()
	if err := tx.UpdateRepeater(r); err != nil {
		return err
	}

	nt := &models.Task{
		Title: r.Title, Description: r.Description, Status: models.StatusOpen, Priority: r.Priority,
		ProjectID: r.ProjectID, AreaID: r.AreaID, HeadingID: r.HeadingID, Evening: r.Evening,
		When:       models.When{Kind: models.WhenScheduled, Date: nextStr},
		RepeaterID: &r.ID, OccurrenceDate: nextStr,
	}
	if r.DeadlineOffset != nil {
		nt.Deadline = next.AddDate(0, 0, *r.DeadlineOffset).Format(models.DateLayout)
	}
	if nt.SortKey, err = appendKey(tx, "tasks"); err != nil {
		return err
	}
	if err := tx.AddTask(nt); err != nil {
		return err
	}
	for _, tag := range done.Tags {
		if err := tx.AttachTagToTask(nt.ID, tag.ID); err != nil {
			return err
		}
	}
	ch.add(nt.ID, ChangeCreated)
	return s.record(tx, nt.ID, "created", fmt.Sprintf("Task created: '%s' (next occurrence, %s)", nt.Title, nextStr))
}

// RepeaterService sets, shows and removes the repeat rule of a task.
type RepeaterService struct {
	store storage.Store
	tasks *TaskService
}

// Set makes the task repeat by rule, or replaces its current rule. A task with
// no day is scheduled for today so it has a first occurrence.
func (s *RepeaterService) Set(taskID int64, rule models.RepeatRule) error {
	return s.tasks.run(func(tx storage.Store, ch *changes) error {
		t, err := s.tasks.get(tx, taskID)
		if err != nil {
			return err
		}
		if err := s.tasks.attachRepeater(tx, t, rule); err != nil {
			return err
		}
		if err := s.tasks.save(tx, t, func(*models.Task) {}); err != nil {
			return err
		}
		ch.add(taskID, ChangeUpdated)
		return nil
	})
}

// ForTask returns the task's Repeater, or nil when it does not repeat.
func (s *RepeaterService) ForTask(taskID int64) (*models.Repeater, error) {
	t, err := s.tasks.get(s.store, taskID)
	if err != nil {
		return nil, err
	}
	if t.RepeaterID == nil {
		return nil, nil
	}
	r, err := s.store.GetRepeater(*t.RepeaterID)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil
	}
	return r, err
}

// Remove stops the series. The tasks already generated stay as ordinary tasks.
func (s *RepeaterService) Remove(taskID int64) error {
	return s.tasks.run(func(tx storage.Store, ch *changes) error {
		t, err := s.tasks.get(tx, taskID)
		if err != nil {
			return err
		}
		if t.RepeaterID == nil {
			return nil
		}
		if err := tx.DeleteRepeater(*t.RepeaterID); err != nil {
			return err
		}
		ch.add(taskID, ChangeUpdated)
		return nil
	})
}
