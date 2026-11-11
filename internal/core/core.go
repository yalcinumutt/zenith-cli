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
	Areas     *AreaService
	Projects  *ProjectService
	Headings  *HeadingService
	Habits    *HabitService
	Tags      *TagService
	Lists     *ListService
	Repeaters *RepeaterService

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
	tasks := &TaskService{store: store, clock: clock, notifier: notifier, bus: b}
	return &Core{
		Tasks:     tasks,
		Areas:     &AreaService{store: store, clock: clock, bus: b, tasks: tasks},
		Projects:  &ProjectService{store: store, clock: clock, bus: b, tasks: tasks},
		Headings:  &HeadingService{store: store, clock: clock, bus: b, tasks: tasks},
		Reminders: &ReminderService{store: store, clock: clock, notifier: notifier, bus: b},
		Habits:    &HabitService{store: store, bus: b},
		Tags:      &TagService{store: store, bus: b},
		Lists:     &ListService{store: store, clock: clock},
		Repeaters: &RepeaterService{store: store, tasks: tasks},
		store:     store,
		bus:       b,
	}
}
