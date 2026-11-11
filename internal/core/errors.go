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

// ErrOpenSubtasks matches *OpenSubtasksError: a task cannot be completed while
// it still has open Subtasks (ADR 0005).
var ErrOpenSubtasks = errors.New("task has open subtasks")

// OpenSubtasksError reports how many open Subtasks block a completion.
type OpenSubtasksError struct {
	TaskID int64
	Count  int
}

func (e *OpenSubtasksError) Error() string {
	noun := "subtasks"
	if e.Count == 1 {
		noun = "subtask"
	}
	return fmt.Sprintf("task %d has %d open %s; complete or cancel them first", e.TaskID, e.Count, noun)
}

func (e *OpenSubtasksError) Is(target error) bool { return target == ErrOpenSubtasks }

// ErrCycle is returned when a nesting change would make a task its own ancestor.
var ErrCycle = errors.New("subtask cycle")

// CycleError names the rejected nesting.
type CycleError struct {
	TaskID, ParentID int64
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("task %d cannot go under task %d: it would become its own ancestor", e.TaskID, e.ParentID)
}

func (e *CycleError) Is(target error) bool { return target == ErrCycle }

// ErrInvalid is returned for requests the rules forbid, with a clear message.
var ErrInvalid = errors.New("invalid request")

type invalidError string

func (e invalidError) Error() string        { return string(e) }
func (e invalidError) Is(target error) bool { return target == ErrInvalid }
