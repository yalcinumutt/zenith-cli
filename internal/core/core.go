// Package core owns Zenith's business rules (ADR 0002). The CLI, TUI and
// desktop shell are thin front-ends over it; the store below it is plain
// persistence. Core imports no Cobra, Bubble Tea, Lipgloss or Wails.
package core

import (
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// Clock supplies the current time. Tests inject a fixed one.
type Clock interface {
	Now() time.Time
}

// SystemClock is the real wall clock.
type SystemClock struct{}

func (SystemClock) Now() time.Time { return time.Now() }

// Notifier delivers a user-visible alert. Implementations per OS arrive with
// the reminders work; the core only depends on this seam.
type Notifier interface {
	Notify(title, message string) error
}

// NopNotifier drops every notification.
type NopNotifier struct{}

func (NopNotifier) Notify(string, string) error { return nil }

// Core bundles the services over one store, clock and notifier.
type Core struct {
	Tasks     *TaskService
	Reminders *ReminderService

	store storage.Store
	bus   *bus
}

// New builds a Core. A nil clock or notifier falls back to the system clock
// and a no-op notifier.
func New(store storage.Store, clock Clock, notifier Notifier) *Core {
	if clock == nil {
		clock = SystemClock{}
	}
	if notifier == nil {
		notifier = NopNotifier{}
	}
	b := &bus{}
	return &Core{
		Tasks:     &TaskService{store: store, clock: clock, notifier: notifier, bus: b},
		Reminders: &ReminderService{store: store, clock: clock, notifier: notifier, bus: b},
		store:     store,
		bus:       b,
	}
}
