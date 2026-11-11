package core_test

import (
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

type countingNotifier struct {
	mu   sync.Mutex
	sent []string
	err  error
}

func (n *countingNotifier) Notify(title, msg string) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.sent = append(n.sent, msg)
	return n.err
}

func (n *countingNotifier) count() int {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.sent)
}

func reminderCore(t *testing.T, path string, n core.Notifier, clock core.Clock) *core.Core {
	t.Helper()
	st, err := storage.NewSQLiteStoreAtPath(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return core.New(st, clock, n)
}

func TestTickDeliversDueReminderOnce(t *testing.T) {
	clock := &fakeClock{now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	n := &countingNotifier{}
	c := reminderCore(t, filepath.Join(t.TempDir(), "r.db"), n, clock)
	var due []core.Event
	c.Subscribe(func(e core.Event) {
		if _, ok := e.(core.ReminderDue); ok {
			due = append(due, e)
		}
	})

	r, err := c.Reminders.Add("stand up", 10*time.Minute, nil)
	if err != nil {
		t.Fatal(err)
	}

	if fired, err := c.Tick(clock.now.Add(5 * time.Minute)); err != nil || len(fired) != 0 {
		t.Fatalf("early tick fired %v, err %v", fired, err)
	}
	fired, err := c.Tick(clock.now.Add(10 * time.Minute))
	if err != nil || len(fired) != 1 || fired[0].ID != r.ID {
		t.Fatalf("tick = %v, %v", fired, err)
	}
	if fired, _ := c.Tick(clock.now.Add(time.Hour)); len(fired) != 0 {
		t.Errorf("second tick fired again: %v", fired)
	}
	if n.count() != 1 || n.sent[0] != "stand up" {
		t.Errorf("notifications = %v, want one 'stand up'", n.sent)
	}
	if len(due) != 1 {
		t.Errorf("ReminderDue events = %d, want 1", len(due))
	}
	if p, _ := c.Reminders.Pending(); len(p) != 0 {
		t.Errorf("pending after fire = %v", p)
	}
}

func TestTwoProcessesNeverFireTheSameReminderTwice(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")
	clock := &fakeClock{now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	n := &countingNotifier{}
	a := reminderCore(t, path, n, clock)
	b := reminderCore(t, path, n, clock) // a second connection stands in for another process

	for i := 0; i < 20; i++ {
		if _, err := a.Reminders.Add("r", time.Minute, nil); err != nil {
			t.Fatal(err)
		}
	}
	later := clock.now.Add(time.Hour)

	var wg sync.WaitGroup
	for _, c := range []*core.Core{a, b} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 5; i++ {
				_, _ = c.Tick(later)
			}
		}()
	}
	wg.Wait()
	if n.count() != 20 {
		t.Errorf("delivered %d notifications for 20 reminders", n.count())
	}
}

func TestNotifierFailureIsReportedAndNotRetried(t *testing.T) {
	boom := errors.New("no notification daemon")
	clock := &fakeClock{now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	n := &countingNotifier{err: boom}
	c := reminderCore(t, filepath.Join(t.TempDir(), "r.db"), n, clock)
	_, _ = c.Reminders.Add("x", time.Minute, nil)

	fired, err := c.Tick(clock.now.Add(time.Hour))
	if len(fired) != 1 || !errors.Is(err, boom) {
		t.Fatalf("fired=%v err=%v", fired, err)
	}
	if fired, err := c.Tick(clock.now.Add(2 * time.Hour)); len(fired) != 0 || err != nil {
		t.Errorf("retried: fired=%v err=%v", fired, err)
	}
}

func TestAddReminderForMissingTaskIsNotFound(t *testing.T) {
	c, _ := newCore(t)
	id := int64(7)
	if _, err := c.Reminders.Add("x", time.Minute, &id); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}
