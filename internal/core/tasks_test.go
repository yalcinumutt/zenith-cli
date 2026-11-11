package core_test

import (
	"context"
	"errors"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

type fakeClock struct{ now time.Time }

func (c *fakeClock) Now() time.Time          { return c.now }
func (c *fakeClock) advance(d time.Duration) { c.now = c.now.Add(d) }

type fakeNotifier struct{ sent []string }

func (n *fakeNotifier) Notify(title, msg string) error { n.sent = append(n.sent, msg); return nil }

func newCore(t *testing.T) (*core.Core, *fakeClock) {
	t.Helper()
	st, err := storage.NewSQLiteStoreAtPath(filepath.Join(t.TempDir(), "core.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	clock := &fakeClock{now: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)}
	return core.New(st, clock, &fakeNotifier{}), clock
}

func actions(t *testing.T, c *core.Core, id int64) []string {
	t.Helper()
	h, err := c.Tasks.History(id)
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for i := len(h) - 1; i >= 0; i-- { // oldest first
		out = append(out, h[i].Action)
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestAddRecordsCreatedHistory(t *testing.T) {
	c, _ := newCore(t)
	task, err := c.Tasks.Add(core.AddTaskInput{Title: "write core", Priority: models.PriorityHigh})
	if err != nil {
		t.Fatal(err)
	}
	if task.Status != core.StatusTodo || task.ID == 0 {
		t.Fatalf("unexpected task %+v", task)
	}
	if got := actions(t, c, task.ID); !eq(got, []string{"created"}) {
		t.Errorf("history = %v", got)
	}
}

func TestNotFoundIsReportedByEveryOperation(t *testing.T) {
	c, _ := newCore(t)
	ops := map[string]func() error{
		"get":      func() error { _, err := c.Tasks.Get(42); return err },
		"complete": func() error { return c.Tasks.Complete(42) },
		"reopen":   func() error { return c.Tasks.Reopen(42) },
		"move":     func() error { return c.Tasks.MoveToProject(42, 1) },
		"plan":     func() error { return c.Tasks.Plan(42, time.Now()) },
		"delete":   func() error { return c.Tasks.Delete(42) },
		"tag":      func() error { return c.Tasks.Tag(42, "x", "") },
		"start":    func() error { return c.Tasks.StartTimer(42) },
		"stop":     func() error { return c.Tasks.StopTimer(42) },
	}
	for name, op := range ops {
		err := op()
		if !errors.Is(err, core.ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
			continue
		}
		if err.Error() != "task with ID 42 not found" {
			t.Errorf("%s: message = %q", name, err.Error())
		}
	}
}

func TestTimerHistoryAndDuration(t *testing.T) {
	c, clock := newCore(t)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "focus"})

	if err := c.Tasks.StartTimer(task.ID); err != nil {
		t.Fatal(err)
	}
	clock.advance(90 * time.Second)
	if err := c.Tasks.StopTimer(task.ID); err != nil {
		t.Fatal(err)
	}

	got, _ := c.Tasks.Get(task.ID)
	if got.IsRunning || got.TotalTime != 90 {
		t.Errorf("IsRunning=%v TotalTime=%d, want false/90", got.IsRunning, got.TotalTime)
	}
	if a := actions(t, c, task.ID); !eq(a, []string{"created", "started_timer", "stopped_timer"}) {
		t.Errorf("history = %v", a)
	}
	h, _ := c.Tasks.History(task.ID)
	if want := "Stopped tracking time. Duration: 0h 1m 30s"; h[0].Details != want {
		t.Errorf("details = %q, want %q", h[0].Details, want)
	}
	if !h[0].Timestamp.Equal(clock.now) {
		t.Errorf("timestamp = %v, want injected clock %v", h[0].Timestamp, clock.now)
	}
}

func TestStartingAgainClosesTheOpenLog(t *testing.T) {
	c, clock := newCore(t)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "restart"})
	_ = c.Tasks.StartTimer(task.ID)
	clock.advance(60 * time.Second)
	_ = c.Tasks.StartTimer(task.ID)
	clock.advance(30 * time.Second)
	_ = c.Tasks.StopTimer(task.ID)

	got, _ := c.Tasks.Get(task.ID)
	if got.TotalTime != 90 || got.IsRunning {
		t.Errorf("TotalTime=%d IsRunning=%v, want 90/false", got.TotalTime, got.IsRunning)
	}
}

func TestStopWithoutTimerIsANoOp(t *testing.T) {
	c, _ := newCore(t)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "idle"})
	if err := c.Tasks.StopTimer(task.ID); err != nil {
		t.Fatal(err)
	}
	if a := actions(t, c, task.ID); !eq(a, []string{"created"}) {
		t.Errorf("history = %v", a)
	}
}

func TestCompleteStopsRunningTimerAndRecordsStatusChange(t *testing.T) {
	c, clock := newCore(t)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "ship"})
	_ = c.Tasks.StartTimer(task.ID)
	clock.advance(10 * time.Second)

	if err := c.Tasks.Complete(task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := c.Tasks.Get(task.ID)
	if got.Status != core.StatusDone || got.IsRunning || got.TotalTime != 10 {
		t.Errorf("got %+v", got)
	}
	if !got.UpdatedAt.Equal(clock.now) {
		t.Errorf("UpdatedAt = %v, want %v", got.UpdatedAt, clock.now)
	}
	want := []string{"created", "started_timer", "stopped_timer", "status_changed"}
	if a := actions(t, c, task.ID); !eq(a, want) {
		t.Errorf("history = %v, want %v", a, want)
	}

	if err := c.Tasks.Reopen(task.ID); err != nil {
		t.Fatal(err)
	}
	got, _ = c.Tasks.Get(task.ID)
	if got.Status != core.StatusTodo {
		t.Errorf("status after reopen = %q", got.Status)
	}
}

func TestEditsRecordEditedHistory(t *testing.T) {
	c, clock := newCore(t)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "plan me"})
	day := clock.now.Truncate(24 * time.Hour)

	if err := c.Tasks.Plan(task.ID, day); err != nil {
		t.Fatal(err)
	}
	if err := c.Tasks.MoveToProject(task.ID, 1); err != nil {
		t.Fatal(err)
	}
	got, _ := c.Tasks.Get(task.ID)
	if got.PlannedDate == nil || got.ProjectID == nil || *got.ProjectID != 1 {
		t.Errorf("got %+v", got)
	}
	if a := actions(t, c, task.ID); !eq(a, []string{"created", "edited", "edited"}) {
		t.Errorf("history = %v", a)
	}
}

func TestTagAndDelete(t *testing.T) {
	c, _ := newCore(t)
	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "tagged"})
	if err := c.Tasks.Tag(task.ID, "urgent", "#f00"); err != nil {
		t.Fatal(err)
	}
	got, _ := c.Tasks.Get(task.ID)
	if len(got.Tags) != 1 || got.Tags[0].Name != "urgent" {
		t.Errorf("tags = %+v", got.Tags)
	}
	if err := c.Tasks.Delete(task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Tasks.Get(task.ID); !errors.Is(err, core.ErrNotFound) {
		t.Errorf("after delete err = %v", err)
	}
}

func collect(c *core.Core) (get func() []core.Event) {
	var mu sync.Mutex
	var got []core.Event
	c.Subscribe(func(e core.Event) {
		mu.Lock()
		got = append(got, e)
		mu.Unlock()
	})
	return func() []core.Event {
		mu.Lock()
		defer mu.Unlock()
		out := got
		got = nil
		return out
	}
}

func TestEveryMutationPublishesAnEvent(t *testing.T) {
	c, _ := newCore(t)
	events := collect(c)

	task, _ := c.Tasks.Add(core.AddTaskInput{Title: "evented"})
	if got := events(); len(got) != 2 {
		t.Fatalf("add: got %d events, want 2", len(got))
	} else if tc, ok := got[0].(core.TaskChanged); !ok || tc.Kind != core.ChangeCreated || tc.TaskID != task.ID {
		t.Errorf("add: first event = %#v", got[0])
	}
	steps := []struct {
		name string
		do   func() error
		kind core.ChangeKind
	}{
		{"start", func() error { return c.Tasks.StartTimer(task.ID) }, core.ChangeTimer},
		{"stop", func() error { return c.Tasks.StopTimer(task.ID) }, core.ChangeTimer},
		{"tag", func() error { return c.Tasks.Tag(task.ID, "t", "") }, core.ChangeTagged},
		{"plan", func() error { return c.Tasks.Plan(task.ID, time.Now()) }, core.ChangeUpdated},
		{"move", func() error { return c.Tasks.MoveToProject(task.ID, 1) }, core.ChangeUpdated},
		{"complete", func() error { return c.Tasks.Complete(task.ID) }, core.ChangeStatus},
		{"reopen", func() error { return c.Tasks.Reopen(task.ID) }, core.ChangeStatus},
		{"delete", func() error { return c.Tasks.Delete(task.ID) }, core.ChangeDeleted},
	}
	for _, s := range steps {
		if err := s.do(); err != nil {
			t.Fatalf("%s: %v", s.name, err)
		}
		got := events()
		if len(got) != 2 {
			t.Fatalf("%s: got %d events, want 2: %v", s.name, len(got), got)
		}
		if tc, ok := got[0].(core.TaskChanged); !ok || tc.TaskID != task.ID || tc.Kind != s.kind {
			t.Errorf("%s: first event = %#v, want TaskChanged{%d,%s}", s.name, got[0], task.ID, s.kind)
		}
		if li, ok := got[1].(core.ListsInvalidated); !ok || len(li.Keys) != 1 || li.Keys[0] != core.ListTasks {
			t.Errorf("%s: second event = %#v", s.name, got[1])
		}
	}
}

func TestFailedMutationPublishesNothing(t *testing.T) {
	c, _ := newCore(t)
	events := collect(c)
	if err := c.Tasks.Complete(99); err == nil {
		t.Fatal("expected not found")
	}
	if got := events(); len(got) != 0 {
		t.Errorf("events = %v, want none", got)
	}
}

func TestUnsubscribeStopsDelivery(t *testing.T) {
	c, _ := newCore(t)
	n := 0
	unsub := c.Subscribe(func(core.Event) { n++ })
	unsub()
	_, _ = c.Tasks.Add(core.AddTaskInput{Title: "quiet"})
	if n != 0 {
		t.Errorf("got %d events after unsubscribe", n)
	}
}

func TestExternalWriteProducesExternalChange(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shared.db")
	ours, err := storage.NewSQLiteStoreAtPath(path)
	if err != nil {
		t.Fatal(err)
	}
	defer ours.Close()
	c := core.New(ours, nil, nil)

	ext := make(chan struct{}, 4)
	c.Subscribe(func(e core.Event) {
		if _, ok := e.(core.ExternalChange); ok {
			ext <- struct{}{}
		}
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = c.WatchExternal(ctx, 50*time.Millisecond) }()
	time.Sleep(150 * time.Millisecond)

	// Our own write must not look external.
	if _, err := c.Tasks.Add(core.AddTaskInput{Title: "mine"}); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ext:
		t.Fatal("own write reported as external change")
	case <-time.After(300 * time.Millisecond):
	}

	// A second connection stands in for the CLI in another process.
	other, err := storage.NewSQLiteStoreAtPath(path)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	if err := core.New(other, nil, nil).Tasks.Complete(1); err != nil {
		t.Fatal(err)
	}
	select {
	case <-ext:
	case <-time.After(2 * time.Second):
		t.Fatal("no ExternalChange within 2s of another connection's write")
	}
}
