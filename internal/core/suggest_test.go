package core_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

type fakeBusy []core.Interval

func (f fakeBusy) Busy(context.Context, string) ([]core.Interval, error) { return f, nil }

type fakeAdvisor struct {
	props []core.Proposal
	err   error
	seen  core.DayInput
}

func (f *fakeAdvisor) Advise(_ context.Context, in core.DayInput) (string, []core.Proposal, error) {
	f.seen = in
	return "ai says hi", f.props, f.err
}

func opts(use bool) core.SuggestOptions {
	return core.SuggestOptions{Date: "2026-03-01", WorkStart: 9 * 60, WorkEnd: 17 * 60, UseAI: use, NowMin: 0}
}

func est(t *testing.T, c *core.Core, id int64, m int) {
	t.Helper()
	if err := c.Tasks.SetEstimate(id, m); err != nil {
		t.Fatal(err)
	}
}

func TestHeuristicOrdersByUrgencyAndAvoidsBusyTime(t *testing.T) {
	c, _ := newCore(t)
	c.Planning.SetBusySource(fakeBusy{{Title: "Standup", StartMin: 9 * 60, EndMin: 10 * 60}})
	low := add(t, c, core.AddTaskInput{Title: "low", When: sched("2026-03-01")})
	high := add(t, c, core.AddTaskInput{Title: "high", When: sched("2026-03-01"), Priority: models.PriorityHigh})
	due := add(t, c, core.AddTaskInput{Title: "due", When: sched("2026-03-01"), Deadline: "2026-03-01"})
	est(t, c, low.ID, 60)
	est(t, c, high.ID, 45)
	est(t, c, due.ID, 30)

	sg, err := c.Planning.Suggest(bg, opts(false))
	if err != nil {
		t.Fatal(err)
	}
	if sg.Source != "heuristic" || len(sg.Unplaced) != 0 {
		t.Fatalf("%+v", sg)
	}
	// Free time starts at 10:00 (after standup): due, then high, then low, back to back.
	want := []core.Proposal{
		{TaskID: due.ID, StartMin: 600, DurationMin: 30, Reason: "Due today"},
		{TaskID: high.ID, StartMin: 630, DurationMin: 45, Reason: "High priority"},
		{TaskID: low.ID, StartMin: 675, DurationMin: 60}, // 11:15: 45 minutes is exactly three 15-minute cells
	}
	if !reflect.DeepEqual(sg.Proposals, want) {
		t.Fatalf("got  %+v\nwant %+v", sg.Proposals, want)
	}
}

func TestHeuristicSkipsExistingBlocksAndReportsUnplaced(t *testing.T) {
	c, _ := newCore(t)
	a := add(t, c, core.AddTaskInput{Title: "a", When: sched("2026-03-01")})
	b := add(t, c, core.AddTaskInput{Title: "b", When: sched("2026-03-01")})
	big := add(t, c, core.AddTaskInput{Title: "big", When: sched("2026-03-01")})
	est(t, c, a.ID, 60)
	est(t, c, b.ID, 60)
	est(t, c, big.ID, 600) // longer than the whole working day
	if _, err := c.Planning.AddBlock(a.ID, "2026-03-01", 9*60, 60); err != nil {
		t.Fatal(err) // a is fully placed already
	}
	sg, _ := c.Planning.Suggest(bg, opts(false))
	if len(sg.Proposals) != 1 || sg.Proposals[0].TaskID != b.ID || sg.Proposals[0].StartMin != 600 {
		t.Fatalf("proposals = %+v", sg.Proposals)
	}
	if !reflect.DeepEqual(sg.Unplaced, []int64{big.ID}) {
		t.Fatalf("unplaced = %v", sg.Unplaced)
	}
}

func TestHeuristicStartsAfterNowToday(t *testing.T) {
	c, _ := newCore(t)
	a := add(t, c, core.AddTaskInput{Title: "a", When: sched("2026-03-01")})
	est(t, c, a.ID, 30)
	o := opts(false)
	o.NowMin = 13*60 + 10 // 13:10 -> first free quarter is 13:15
	sg, _ := c.Planning.Suggest(bg, o)
	if len(sg.Proposals) != 1 || sg.Proposals[0].StartMin != 13*60+15 {
		t.Fatalf("%+v", sg.Proposals)
	}
}

func TestApplyIsAllOrNothing(t *testing.T) {
	c, _ := newCore(t)
	a := add(t, c, core.AddTaskInput{Title: "a", When: sched("2026-03-01")})
	b := add(t, c, core.AddTaskInput{Title: "b", When: sched("2026-03-01")})
	good := core.Proposal{TaskID: a.ID, StartMin: 540, DurationMin: 30}
	if _, err := c.Planning.Apply("2026-03-01", []core.Proposal{good, {TaskID: b.ID, StartMin: 555, DurationMin: 30}}); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("overlapping proposals: %v", err)
	}
	if p, _ := c.Planning.Plan(bg, "2026-03-01"); len(p.Blocks) != 0 {
		t.Fatalf("a failed apply must leave nothing behind: %+v", p.Blocks)
	}
	blocks, err := c.Planning.Apply("2026-03-01", []core.Proposal{good, {TaskID: b.ID, StartMin: 570, DurationMin: 30}})
	if err != nil || len(blocks) != 2 {
		t.Fatalf("apply: %v %v", blocks, err)
	}
	// Applying the same suggestion again must refuse rather than double-book.
	if _, err := c.Planning.Apply("2026-03-01", []core.Proposal{good}); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("re-apply: %v", err)
	}
}

func TestAIProposalsAreValidated(t *testing.T) {
	c, _ := newCore(t)
	c.Planning.SetBusySource(fakeBusy{{StartMin: 12 * 60, EndMin: 13 * 60}})
	a := add(t, c, core.AddTaskInput{Title: "a", When: sched("2026-03-01")})
	b := add(t, c, core.AddTaskInput{Title: "b", When: sched("2026-03-01")})
	est(t, c, a.ID, 60)
	est(t, c, b.ID, 30)
	adv := &fakeAdvisor{props: []core.Proposal{
		{TaskID: a.ID, StartMin: 9 * 60, DurationMin: 60, Reason: "deep work"}, // ok
		{TaskID: b.ID, StartMin: 12*60 + 15, DurationMin: 30},                  // in the calendar event
		{TaskID: 999, StartMin: 14 * 60, DurationMin: 30},                      // unknown task
		{TaskID: b.ID, StartMin: 7 * 60, DurationMin: 30},                      // before working hours
		{TaskID: b.ID, StartMin: 9*60 + 30, DurationMin: 30},                   // overlaps the first proposal
		{TaskID: a.ID, StartMin: 14 * 60, DurationMin: 30},                     // a already has all its minutes
		{TaskID: b.ID, StartMin: 14*60 + 2, DurationMin: 30},                   // off the 5-minute grid
	}}
	c.Planning.SetAdvisor(adv)

	sg, err := c.Planning.Suggest(bg, opts(true))
	if err != nil {
		t.Fatal(err)
	}
	if sg.Source != "ai" || sg.Summary != "ai says hi" {
		t.Fatalf("%+v", sg)
	}
	if len(sg.Proposals) != 1 || sg.Proposals[0].TaskID != a.ID || sg.Proposals[0].Reason != "deep work" {
		t.Fatalf("kept = %+v", sg.Proposals)
	}
	if len(sg.Dropped) != 6 {
		t.Fatalf("dropped = %q", sg.Dropped)
	}
	if !reflect.DeepEqual(sg.Unplaced, []int64{b.ID}) {
		t.Fatalf("unplaced = %v", sg.Unplaced)
	}
	if len(adv.seen.Tasks) != 2 || len(adv.seen.Busy) != 1 {
		t.Fatalf("advisor input = %+v", adv.seen)
	}
}

func TestAIErrorsAndMissingAdvisor(t *testing.T) {
	c, _ := newCore(t)
	add(t, c, core.AddTaskInput{Title: "a", When: sched("2026-03-01")})
	if _, err := c.Planning.Suggest(bg, opts(true)); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("no advisor: %v", err)
	}
	boom := errors.New("model offline")
	c.Planning.SetAdvisor(&fakeAdvisor{err: boom})
	if _, err := c.Planning.Suggest(bg, opts(true)); !errors.Is(err, boom) {
		t.Fatalf("advisor error should pass through: %v", err)
	}
	bad := opts(false)
	bad.WorkEnd = bad.WorkStart
	if _, err := c.Planning.Suggest(bg, bad); !errors.Is(err, core.ErrInvalid) {
		t.Fatalf("empty working hours: %v", err)
	}
}
