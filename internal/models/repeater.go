package models

import (
	"fmt"
	"strings"
	"time"
)

// RepeatKind is the unit of a repeat rule. The format is deliberately small
// (ADR 0006): every N days, every N weeks on given weekdays, or every N months
// on a day-of-month.
type RepeatKind string

const (
	RepeatDays   RepeatKind = "days"
	RepeatWeeks  RepeatKind = "weeks"
	RepeatMonths RepeatKind = "months"
)

// RepeatRule says when occurrences fall.
type RepeatRule struct {
	Kind     RepeatKind `json:"kind"`
	Interval int        `json:"interval"`           // every N units, >= 1
	Weekdays []int      `json:"weekdays,omitempty"` // weeks only: 0=Sunday .. 6=Saturday; empty = the occurrence's own weekday
	MonthDay int        `json:"month_day,omitempty"` // months only: 1..31, clamped to the month's length
}

// String describes the rule for display, e.g. "Every 2 weeks on Mon, Wed".
func (r RepeatRule) String() string {
	every := "Every"
	switch r.Kind {
	case RepeatDays:
		if r.Interval <= 1 {
			return "Every day"
		}
		return fmt.Sprintf("%s %d days", every, r.Interval)
	case RepeatWeeks:
		s := "Every week"
		if r.Interval > 1 {
			s = fmt.Sprintf("Every %d weeks", r.Interval)
		}
		if len(r.Weekdays) > 0 {
			var names []string
			for _, d := range r.Weekdays {
				names = append(names, time.Weekday(d).String()[:3])
			}
			s += " on " + strings.Join(names, ", ")
		}
		return s
	case RepeatMonths:
		s := "Every month"
		if r.Interval > 1 {
			s = fmt.Sprintf("Every %d months", r.Interval)
		}
		if r.MonthDay > 0 {
			s += fmt.Sprintf(" on day %d", r.MonthDay)
		}
		return s
	}
	return "Repeats"
}

// Repeater is the rule and template behind a repeating task. Each occurrence
// it produces is an ordinary Task pointing back at it; the template follows the
// most recently completed occurrence.
type Repeater struct {
	ID             int64      `json:"id"`
	UUID           string     `json:"uuid"`
	Rule           RepeatRule `json:"rule"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	ProjectID      *int64     `json:"project_id"`
	AreaID         *int64     `json:"area_id"`
	HeadingID      *int64     `json:"heading_id"`
	Priority       Priority   `json:"priority"`
	Evening        bool       `json:"evening"`
	DeadlineOffset *int       `json:"deadline_offset"` // days from the occurrence date to its Deadline
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}
