package models

import "time"

type TaskTimeLog struct {
	ID          int64      `json:"id"`
	TaskID      int64      `json:"task_id"`
	StartTime   time.Time  `json:"start_time"`
	EndTime     *time.Time `json:"end_time"`
	Description string     `json:"description"`
}
