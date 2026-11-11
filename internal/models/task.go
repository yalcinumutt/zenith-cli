package models

import "time"

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

// Task and Project statuses (glossary: Status). "In progress" is deliberately
// absent: it is derived from a running Timer, see Task.InProgress.
const (
	StatusOpen      = "open"
	StatusCompleted = "completed"
	StatusCancelled = "cancelled"
)

// WhenKind says when the user intends to work on a task (glossary: When).
type WhenKind string

const (
	WhenAnytime   WhenKind = "anytime"   // not scheduled
	WhenScheduled WhenKind = "scheduled" // a specific day; "Today" is a scheduled day that has arrived
	WhenSomeday   WhenKind = "someday"   // parked, no intention yet
)

// DateLayout is the only date format: plain calendar days, no time zone.
const DateLayout = "2006-01-02"

// When is a sum type: Date is set only for WhenScheduled. Distinct from a
// task's Deadline.
type When struct {
	Kind WhenKind `json:"kind"`
	Date string   `json:"date,omitempty"`
}

type Task struct {
	ID             int64     `json:"id"`
	UUID           string    `json:"uuid"` // sync identity (ADR 0003); ID stays the local handle
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	Status         string    `json:"status"` // open, completed, cancelled
	Priority       Priority  `json:"priority"`
	ProjectID      *int64    `json:"project_id"`
	AreaID         *int64    `json:"area_id"`    // set only for tasks filed directly in an Area
	HeadingID      *int64    `json:"heading_id"` // set only inside a Project
	ParentID       *int64    `json:"parent_id"`  // container Task when this is a Subtask (ADR 0005)
	When           When      `json:"when"`
	Deadline       string    `json:"deadline"` // YYYY-MM-DD, empty when none
	Evening        bool      `json:"evening"`  // "This Evening" within Today
	Tags           []Tag     `json:"tags"`
	IsRunning      bool      `json:"is_running"`
	TotalTime      int64     `json:"total_time"`      // in seconds
	RepeaterID     *int64    `json:"repeater_id"`     // the Repeater this task is an occurrence of
	OccurrenceDate string    `json:"occurrence_date"` // the day this occurrence was generated for
	SortKey        string    `json:"sort_key"`        // fractional index; order is (SortKey, UUID)
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// InProgress is derived, never stored: an open task with a running Timer.
func (t Task) InProgress() bool { return t.Status == StatusOpen && t.IsRunning }
