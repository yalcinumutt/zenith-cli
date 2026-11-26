package models

import "time"

// TimeBlock is a slice of one calendar day a user has set aside to work on a
// Task (glossary: Time Block). It is separate from the Task's When, which stays
// a plain date, and from the Timer, which records time actually spent.
type TimeBlock struct {
	ID          int64     `json:"id"`
	UUID        string    `json:"uuid"` // sync identity (ADR 0003); ID stays the local handle
	TaskID      int64     `json:"task_id"`
	Date        string    `json:"date"`         // YYYY-MM-DD
	StartMin    int       `json:"start_min"`    // minutes after local midnight, 0..1439
	DurationMin int       `json:"duration_min"` // minutes, >= 1; start+duration never passes midnight
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EndMin is the minute the block ends at (exclusive).
func (b TimeBlock) EndMin() int { return b.StartMin + b.DurationMin }
