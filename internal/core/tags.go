package core

import (
	"strings"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// TagService owns tag identity. Tags are unique by case-insensitive name, and
// that is enforced here rather than by a database constraint so that two
// devices creating "Work" and "work" offline can later be merged
// deterministically (ADR 0010).
type TagService struct {
	store storage.Store
	bus   *bus
}

// NormalizeTagName is the identity of a tag: trimmed and case-folded.
func NormalizeTagName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// findOrCreate returns the existing tag with the same normalized name (the
// oldest, if legacy data holds several), or creates it. A non-empty color
// updates an existing tag's color.
func findOrCreateTag(tx storage.Store, name, color string) (*models.Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, invalidError("tag name cannot be empty")
	}
	all, err := tx.GetTags()
	if err != nil {
		return nil, err
	}
	key := NormalizeTagName(name)
	for i := range all {
		if NormalizeTagName(all[i].Name) == key {
			t := &all[i]
			if color != "" && color != t.Color {
				t.Color = color
				if err := tx.UpdateTag(t); err != nil {
					return nil, err
				}
			}
			return t, nil
		}
	}
	t := &models.Tag{Name: name, Color: color}
	if err := tx.AddTag(t); err != nil {
		return nil, err
	}
	return t, nil
}

// FindOrCreate returns the tag for name, creating it when no tag matches in any case.
func (s *TagService) FindOrCreate(name, color string) (*models.Tag, error) {
	var tag *models.Tag
	err := s.store.WithTx(func(tx storage.Store) error {
		var err error
		tag, err = findOrCreateTag(tx, name, color)
		return err
	})
	if err != nil {
		return nil, err
	}
	s.bus.publish(ListsInvalidated{Keys: []string{ListTags}})
	return tag, nil
}

// List returns all tags, oldest first.
func (s *TagService) List() ([]models.Tag, error) { return s.store.GetTags() }
