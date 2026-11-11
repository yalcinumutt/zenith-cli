package cmd

import (
	"fmt"
	"strconv"
)

// parseID reads a positional integer handle.
func parseID(what, s string) (int64, error) {
	id, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", what, err)
	}
	return id, nil
}

// parseOptionalID reads an integer handle, where "none" means no container.
func parseOptionalID(what, s string) (*int64, error) {
	if s == "none" {
		return nil, nil
	}
	id, err := parseID(what, s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}
