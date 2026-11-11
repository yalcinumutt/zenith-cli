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

type Task struct {
	ID          int64      `json:"id"`
	UUID        string     `json:"uuid"` // sync identity (ADR 0003); ID stays the local handle
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"` // open, completed, cancelled
	Priority    Priority   `json:"priority"`
	ProjectID   *int64     `json:"project_id"`
	AreaID      *int64     `json:"area_id"`    // set only for tasks filed directly in an Area
	HeadingID   *int64     `json:"heading_id"` // set only inside a Project
	ParentID    *int64     `json:"parent_id"`  // container Task when this is a Subtask (ADR 0005)
	DueDate     *time.Time `json:"due_date"`
	PlannedDate *time.Time `json:"planned_date"`
	Tags        []Tag      `json:"tags"`
	IsRunning   bool       `json:"is_running"`
	TotalTime   int64      `json:"total_time"` // in seconds
	Recurring   string     `json:"recurring"`  // daily, weekly, monthly, none
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// InProgress is derived, never stored: an open task with a running Timer.
func (t Task) InProgress() bool { return t.Status == StatusOpen && t.IsRunning }
