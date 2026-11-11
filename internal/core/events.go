package core

import (
	"context"
	"sync"
	"time"
)

// Events are the core's public change notifications (ADR 0011). Front-ends
// subscribe once and react: the TUI refetches, the desktop shell forwards them
// to the UI's query invalidation, a future tray or plugin host listens too.
//
// Rules for this surface:
//   - Payloads carry ids and a coarse kind, never whole entities. A subscriber
//     that needs the data asks the core.
//   - The set is additive-only: new event types and new fields may appear, but
//     existing types, fields and ChangeKind/list-key values are never renamed,
//     repurposed or removed. Subscribers must ignore events they don't know.
//   - Entities are identified by their integer handle for now. The sync uuid
//     (ADR 0003) arrives as an added field when the two-ID work lands.
//   - Handlers run synchronously on the goroutine that caused the change, after
//     it has committed. They must return quickly and must not call back into a
//     mutating core method; hand work off to a channel or goroutine instead.
type Event interface {
	isEvent()
}

// ChangeKind is the coarse reason a task changed.
type ChangeKind string

const (
	ChangeCreated ChangeKind = "created" // the task was added
	ChangeUpdated ChangeKind = "updated" // fields other than status changed
	ChangeStatus  ChangeKind = "status"  // completed or reopened
	ChangeTimer   ChangeKind = "timer"   // time tracking started or stopped
	ChangeTagged  ChangeKind = "tagged"  // a tag was attached
	ChangeDeleted ChangeKind = "deleted" // the task was removed
)

// List keys carried by ListsInvalidated.
const (
	// ListTasks covers every task-derived view. Finer keys (one per Things-style
	// list) are added alongside it when the derived lists exist.
	ListTasks = "tasks"
)

// TaskChanged is published after a task mutation made through this core.
type TaskChanged struct {
	TaskID int64
	Kind   ChangeKind
}

// ListsInvalidated names the cached lists whose contents may have changed.
type ListsInvalidated struct {
	Keys []string
}

// ReminderDue is published by Tick when a reminder fires in this process.
type ReminderDue struct {
	ReminderID int64
}

// ExternalChange means the database changed outside this process (the CLI, the
// daemon, another app instance). It carries no detail: refetch what you show.
type ExternalChange struct{}

func (TaskChanged) isEvent()      {}
func (ListsInvalidated) isEvent() {}
func (ReminderDue) isEvent()      {}
func (ExternalChange) isEvent()   {}

// bus is a minimal synchronous in-process fan-out.
type bus struct {
	mu   sync.RWMutex
	next int
	subs map[int]func(Event)
}

func (b *bus) subscribe(fn func(Event)) func() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.subs == nil {
		b.subs = map[int]func(Event){}
	}
	id := b.next
	b.next++
	b.subs[id] = fn
	return func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		delete(b.subs, id)
	}
}

func (b *bus) publish(events ...Event) {
	b.mu.RLock()
	handlers := make([]func(Event), 0, len(b.subs))
	for _, fn := range b.subs {
		handlers = append(handlers, fn)
	}
	b.mu.RUnlock()
	for _, e := range events {
		for _, fn := range handlers {
			fn(e)
		}
	}
}

// Subscribe registers fn for every event and returns a function that removes it.
func (c *Core) Subscribe(fn func(Event)) (unsubscribe func()) {
	return c.bus.subscribe(fn)
}

// WatchExternal polls the database's change counter every interval and
// publishes ExternalChange when a different process has written. It blocks
// until ctx is cancelled, so run it in a goroutine. Writes made through this
// process's own connection never trigger it.
func (c *Core) WatchExternal(ctx context.Context, interval time.Duration) error {
	last, err := c.store.DataVersion()
	if err != nil {
		return err
	}
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			v, err := c.store.DataVersion()
			if err != nil {
				continue // transient (busy); try again next tick
			}
			if v != last {
				last = v
				c.bus.publish(ExternalChange{}, ListsInvalidated{Keys: []string{ListTasks}})
			}
		}
	}
}
