package models

import "time"

type ScheduledTimer struct {
	ID          int64     `json:"id"`
	TaskID      *int64    `json:"task_id"`
	Title       string    `json:"title"`
	TriggerTime time.Time `json:"trigger_time"`
	Status      string    `json:"status"` // pending, triggered, dismissed
	CreatedAt   time.Time `json:"created_at"`
}
