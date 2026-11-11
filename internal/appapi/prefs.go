package appapi

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
)

// prefsStore keeps the app's preferences in a small JSON file next to the
// database, so they survive relaunches without relying on web-view storage.
type prefsStore struct {
	mu   sync.Mutex
	path string
}

var defaultPrefs = Prefs{Theme: "system", LastView: "today"}

func (p *prefsStore) load() Prefs {
	p.mu.Lock()
	defer p.mu.Unlock()
	prefs := defaultPrefs
	if p.path == "" {
		return prefs
	}
	if b, err := os.ReadFile(p.path); err == nil {
		_ = json.Unmarshal(b, &prefs)
	}
	if prefs.Theme != "light" && prefs.Theme != "dark" {
		prefs.Theme = "system"
	}
	if prefs.LastView == "" {
		prefs.LastView = defaultPrefs.LastView
	}
	return prefs
}

func (p *prefsStore) save(prefs Prefs) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.path == "" {
		return errors.New("preferences are not available")
	}
	b, err := json.MarshalIndent(prefs, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(p.path), 0o755); err != nil {
		return err
	}
	tmp := p.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, p.path)
}
