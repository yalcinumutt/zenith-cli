package core

import (
	"errors"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/orderkey"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// Manual order (ADR 0007): every orderable row has a fractional-index sort key,
// and rows sort by (key, uuid). Placing a row between two neighbours computes one
// new key and writes only that row, so a reorder cannot disturb its neighbours
// and merges as an ordinary field under last-write-wins.

// appendKey returns a key after every existing row of the table.
func appendKey(tx storage.Store, table string) (string, error) {
	max, err := tx.MaxSortKey(table)
	if err != nil {
		return "", err
	}
	return orderkey.Between(max, "")
}

// keyBetween picks a key between two neighbours' keys. Equal or inverted keys
// (possible after two devices merge) fall back to "just after lo".
func keyBetween(lo, hi string) (string, error) {
	k, err := orderkey.Between(lo, hi)
	if errors.Is(err, orderkey.ErrOrder) {
		return orderkey.Between(lo, "")
	}
	return k, err
}

// Placement describes where a task goes in a single write. AfterID and BeforeID
// are the neighbours in the list being viewed (either may be nil at an edge).
// Heading and Evening are applied in the same write when set.
type Placement struct {
	AfterID, BeforeID *int64
	Heading           *HeadingChange // nil leaves the heading as it is
	Evening           *bool          // nil leaves the evening flag as it is
}

// HeadingChange puts a task under a heading, or takes it out when HeadingID is nil.
type HeadingChange struct {
	HeadingID *int64
}

// Reorder moves the task between two neighbours.
func (s *TaskService) Reorder(id int64, afterID, beforeID *int64) error {
	return s.Place(id, Placement{AfterID: afterID, BeforeID: beforeID})
}

// Place moves a task as one single-row write: its new position, and optionally
// its heading (dragging across headings) and its This Evening flag (dragging
// between the day and evening sections of Today).
func (s *TaskService) Place(id int64, p Placement) error {
	return s.run(func(tx storage.Store, ch *changes) error {
		t, err := s.get(tx, id)
		if err != nil {
			return err
		}
		if p.AfterID != nil || p.BeforeID != nil {
			var lo, hi string
			if p.AfterID != nil {
				a, err := s.get(tx, *p.AfterID)
				if err != nil {
					return err
				}
				lo = a.SortKey
			}
			if p.BeforeID != nil {
				b, err := s.get(tx, *p.BeforeID)
				if err != nil {
					return err
				}
				hi = b.SortKey
			}
			if t.SortKey, err = keyBetween(lo, hi); err != nil {
				return err
			}
		}
		if p.Heading != nil {
			if p.Heading.HeadingID == nil {
				t.HeadingID = nil
			} else {
				h, err := getHeading(tx, *p.Heading.HeadingID)
				if err != nil {
					return err
				}
				t.HeadingID, t.ProjectID, t.AreaID = &h.ID, &h.ProjectID, nil
			}
		}
		if p.Evening != nil {
			if *p.Evening && t.When.Kind != models.WhenScheduled {
				return invalidError("only a scheduled task can go in This Evening")
			}
			t.Evening = *p.Evening
		}
		t.UpdatedAt = s.clock.Now()
		if err := tx.UpdateTask(t); err != nil {
			return err
		}
		ch.add(id, ChangeOrder)
		return nil
	})
}

// reorderKey resolves the new key for a row among siblings of its kind.
func reorderKey(lookup func(id int64) (string, error), afterID, beforeID *int64) (string, error) {
	var lo, hi string
	var err error
	if afterID != nil {
		if lo, err = lookup(*afterID); err != nil {
			return "", err
		}
	}
	if beforeID != nil {
		if hi, err = lookup(*beforeID); err != nil {
			return "", err
		}
	}
	return keyBetween(lo, hi)
}

// Reorder places a heading between two neighbours.
func (s *HeadingService) Reorder(id int64, afterID, beforeID *int64) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		h, err := getHeading(tx, id)
		if err != nil {
			return err
		}
		h.SortKey, err = reorderKey(func(n int64) (string, error) {
			o, err := getHeading(tx, n)
			if err != nil {
				return "", err
			}
			return o.SortKey, nil
		}, afterID, beforeID)
		if err != nil {
			return err
		}
		h.UpdatedAt = s.clock.Now()
		return tx.UpdateHeading(h)
	})
	if err == nil {
		s.announce()
	}
	return err
}

// Reorder places a project between two neighbours.
func (s *ProjectService) Reorder(id int64, afterID, beforeID *int64) error {
	return s.update(id, func(tx storage.Store, p *models.Project) error {
		key, err := reorderKey(func(n int64) (string, error) {
			o, err := s.get(tx, n)
			if err != nil {
				return "", err
			}
			return o.SortKey, nil
		}, afterID, beforeID)
		p.SortKey = key
		return err
	})
}

// Reorder places an area between two neighbours.
func (s *AreaService) Reorder(id int64, afterID, beforeID *int64) error {
	err := s.store.WithTx(func(tx storage.Store) error {
		a, err := getArea(tx, id)
		if err != nil {
			return err
		}
		a.SortKey, err = reorderKey(func(n int64) (string, error) {
			o, err := getArea(tx, n)
			if err != nil {
				return "", err
			}
			return o.SortKey, nil
		}, afterID, beforeID)
		if err != nil {
			return err
		}
		a.UpdatedAt = s.clock.Now()
		return tx.UpdateArea(a)
	})
	if err == nil {
		s.announce()
	}
	return err
}
