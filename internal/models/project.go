package models

import "time"

type Project struct {
	ID          int64     `json:"id"`
	UUID        string    `json:"uuid"` // sync identity (ADR 0003); ID stays the local handle
	AreaID      *int64    `json:"area_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // open, completed, cancelled
	SortKey     string    `json:"sort_key"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Area is a long-lived bucket of responsibility. Areas never complete.
type Area struct {
	ID        int64     `json:"id"`
	UUID      string    `json:"uuid"`
	Name      string    `json:"name"`
	SortKey   string    `json:"sort_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Heading groups Tasks inside one Project. It holds nothing but Tasks and has
// no completion state.
type Heading struct {
	ID        int64     `json:"id"`
	UUID      string    `json:"uuid"`
	ProjectID int64     `json:"project_id"`
	Title     string    `json:"title"`
	SortKey   string    `json:"sort_key"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
