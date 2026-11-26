package core_test

import (
	"context"
	"errors"
	"testing"

	"github.com/yalcinumutt/zenith-cli/internal/core"
)

var bg = context.Background()

func TestPlanListsTodayTasksAndBlocks(t *testing.T) {
	c, _ := newCore(t)
	a := add(t, c, core.AddTaskInput{Title: "write", When: sched("2026-03-01")})
	add(t, c, core.AddTaskInput{Title: "tomorrow", When: sched("2026-03-02")})

	if _, err := c.Planning.AddBlock(a.ID, "2026-03-01", 9*60, 90); err != nil {
		t.Fatal(err)
	}
	p, err := c.Planning.Plan(bg, "2026-03-01")
	if err != nil {
		t.Fatal(err)
	}
	if got := titles(p.Tasks); len(got) != 1 || got[0] != "write" {
		t.Fatalf("tasks = %v", got)
	}
	if len(p.Blocks) != 1 || p.Blocks[0].StartMin != 540 || p.Blocks[0].DurationMin != 90 {
		t.Fatalf("blocks = %+v", p.Blocks)
	}
	tm, _ := c.Planning.Plan(bg, "2026-03-02")
	if got := titles(tm.Tasks); len(got) != 1 || got[0] != "tomorrow" || len(tm.Blocks) != 0 {
		t.Fatalf("tomorrow = %v %v", got, tm.Blocks)
	}
}

func TestBlocksNeverOverlapOrCrossMidnight(t *testing.T) {
	c, _ := newCore(t)
	a := add(t, c, core.AddTaskInput{Title: "a", When: sched("2026-03-01")})
	b := add(t, c, core.AddTaskInput{Title: "b", When: sched("2026-03-01")})
	first, err := c.Planning.AddBlock(a.ID, "2026-03-01", 600, 60) // 10:00-11:00
	if err != nil {
		t.Fatal(err)
	}
	for name, start := range map[string]int{"inside": 630, "overlap-start": 570, "same": 600} {
		if _, err := c.Planning.AddBlock(b.ID, "2026-03-01", start, 60); !errors.Is(err, core.ErrInvalid) {
			t.Errorf("%s: want ErrInvalid, got %v", name, err)
		}
	}
	if _, err := c.Planning.AddBlock(b.ID, "2026-03-01", 660, 30); err != nil { // touching is fine
		t.Fatalf("adjacent block: %v", err)
	}
	if _, err := c.Planning.AddBlock(b.ID, "2026-03-01", 23*60+30, 60); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("past midnight: %v", err)
	}
	if _, err := c.Planning.AddBlock(b.ID, "2026-03-01", 100, 2); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("too short: %v", err)
	}
	// A block may be moved over its own old slot, but not onto another block.
	if err := c.Planning.Move(first.ID, "2026-03-01", 590); err != nil {
		t.Fatalf("shift within own slot: %v", err)
	}
	if err := c.Planning.Move(first.ID, "2026-03-01", 650); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("move onto neighbour: %v", err)
	}
	if err := c.Planning.Resize(first.ID, 120); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("resize into neighbour: %v", err)
	}
}

func TestBlockLifecycleAndTaskRules(t *testing.T) {
	c, _ := newCore(t)
	a := add(t, c, core.AddTaskInput{Title: "a", When: sched("2026-03-01")})
	blk, err := c.Planning.AddBlock(a.ID, "2026-03-01", 480, 30)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Planning.Resize(blk.ID, 45); err != nil {
		t.Fatal(err)
	}
	if err := c.Planning.Move(blk.ID, "2026-03-02", 600); err != nil {
		t.Fatal(err)
	}
	if p, _ := c.Planning.Plan(bg, "2026-03-01"); len(p.Blocks) != 0 {
		t.Fatalf("block should have left the day: %+v", p.Blocks)
	}
	if p, _ := c.Planning.Plan(bg, "2026-03-02"); len(p.Blocks) != 1 || p.Blocks[0].DurationMin != 45 {
		t.Fatalf("moved block = %+v", p.Blocks)
	}
	if err := c.Planning.Remove(blk.ID); err != nil {
		t.Fatal(err)
	}
	if err := c.Planning.Remove(blk.ID); err != nil { // idempotent
		t.Fatal(err)
	}
	if err := c.Planning.Resize(blk.ID, 30); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("resize removed: %v", err)
	}
	if _, err := c.Planning.AddBlock(999, "2026-03-01", 480, 30); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("unknown task: %v", err)
	}
	if err := c.Tasks.Complete(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Planning.AddBlock(a.ID, "2026-03-01", 480, 30); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("completed task: %v", err)
	}
}

func TestBlocksOfDeletedTasksDisappear(t *testing.T) {
	c, _ := newCore(t)
	a := add(t, c, core.AddTaskInput{Title: "a", When: sched("2026-03-01")})
	if _, err := c.Planning.AddBlock(a.ID, "2026-03-01", 480, 30); err != nil {
		t.Fatal(err)
	}
	if err := c.Tasks.Delete(a.ID); err != nil {
		t.Fatal(err)
	}
	if p, _ := c.Planning.Plan(bg, "2026-03-01"); len(p.Blocks) != 0 {
		t.Fatalf("blocks = %+v", p.Blocks)
	}
}

func TestSetEstimate(t *testing.T) {
	c, _ := newCore(t)
	a := add(t, c, core.AddTaskInput{Title: "a"})
	if err := c.Tasks.SetEstimate(a.ID, 45); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(a.ID); got.EstimateMinutes != 45 {
		t.Fatalf("estimate = %d", got.EstimateMinutes)
	}
	if err := c.Tasks.SetEstimate(a.ID, 0); err != nil {
		t.Fatal(err)
	}
	if got, _ := c.Tasks.Get(a.ID); got.EstimateMinutes != 0 {
		t.Fatalf("estimate = %d", got.EstimateMinutes)
	}
	if err := c.Tasks.SetEstimate(a.ID, -1); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("negative: %v", err)
	}
}
