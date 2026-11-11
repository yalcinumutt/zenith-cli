package models

import "time"

type Project struct {
	ID          int64     `json:"id"`
	UUID        string    `json:"uuid"` // sync identity (ADR 0003); ID stays the local handle
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
