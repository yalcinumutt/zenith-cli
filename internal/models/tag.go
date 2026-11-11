package models

import "time"

type Tag struct {
	ID        int64     `json:"id"`
	UUID      string    `json:"uuid"` // sync identity (ADR 0003); ID stays the local handle
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}
