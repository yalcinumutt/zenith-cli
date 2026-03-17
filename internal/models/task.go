package models

import "time"

type Priority int

const (
	PriorityLow Priority = iota
	PriorityMedium
	PriorityHigh
	PriorityCritical
)

type Task struct {
	ID          int64     `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // Todo, InProgress, Done
	Priority    Priority  `json:"priority"`
	ProjectID   *int64    `json:"project_id"`
	DueDate     *time.Time `json:"due_date"`
	Recurring   string    `json:"recurring"` // daily, weekly, monthly, none
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
