package models

import "time"

type TaskHistory struct {
	ID        int64     `json:"id"`
	TaskID    int64     `json:"task_id"`
	Action    string    `json:"action"` // created, status_changed, started_timer, stopped_timer, edited
	Details   string    `json:"details"`
	Timestamp time.Time `json:"timestamp"`
}
