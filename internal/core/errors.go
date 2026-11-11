package core

import (
	"errors"
	"fmt"
)

// ErrNotFound matches every NotFoundError via errors.Is.
var ErrNotFound = errors.New("not found")

// NotFoundError reports a missing entity by its integer handle.
type NotFoundError struct {
	Entity string
	ID     int64
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s with ID %d not found", e.Entity, e.ID)
}

func (e *NotFoundError) Is(target error) bool { return target == ErrNotFound }
