package models

import "time"

type Habit struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Frequency   string    `json:"frequency"` // daily, weekly
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type HabitLog struct {
	ID        int64     `json:"id"`
	HabitID   int64     `json:"habit_id"`
	CompletedAt time.Time `json:"completed_at"`
}
