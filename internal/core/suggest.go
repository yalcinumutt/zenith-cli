package core

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

const (
	// DefaultBlockMinutes sizes a block for a task that has no Estimate.
	DefaultBlockMinutes = 30
	suggestSnap         = 15
	aiSnap              = 5
)

// Proposal is a block the planner suggests. It is only a suggestion until the
// user applies it: nothing here writes to the store.
type Proposal struct {
	TaskID      int64
	StartMin    int
	DurationMin int
	Reason      string
}

// PlanTask is a task to place and how many minutes of it still need a slot.
type PlanTask struct {
	Task    models.Task
	Minutes int
}

// DayInput is everything an Advisor sees about one day. Busy merges calendar
// events and blocks already placed; titles are included for the user's own
// display but an Advisor sending data off the machine should omit them.
type DayInput struct {
	Date      string
	WorkStart int
	WorkEnd   int
	NowMin    int // earliest usable minute today; 0 for other days
	Tasks     []PlanTask
	Busy      []Interval
}

// Advisor proposes placements, typically an AI model. The core validates
// whatever it returns, so a confused or malicious answer cannot produce an
// overlapping, out-of-hours or unknown-task block.
type Advisor interface {
	Advise(ctx context.Context, in DayInput) (summary string, proposals []Proposal, err error)
}

// SetAdvisor wires AI suggestions into planning; nil removes them.
func (s *PlanningService) SetAdvisor(a Advisor) { s.advisor = a }

// HasAdvisor reports whether AI suggestions are available.
func (s *PlanningService) HasAdvisor() bool { return s.advisor != nil }

// SuggestOptions choose the day, the working hours and the engine.
type SuggestOptions struct {
	Date      string
	WorkStart int  // minutes after midnight
	WorkEnd   int  // minutes after midnight, exclusive
	UseAI     bool // ask the Advisor; otherwise use the built-in scheduler
	NowMin    int  // current minute, honoured only when Date is today
}

// Suggestion is a proposed plan for review.
type Suggestion struct {
	Source    string // "heuristic" or "ai"
	Summary   string
	Proposals []Proposal
	Unplaced  []int64  // tasks with no slot
	Dropped   []string // AI proposals rejected, with the reason
}

// Suggest proposes blocks for the tasks that still need time on a day. It reads
// the plan, never writes it.
func (s *PlanningService) Suggest(ctx context.Context, o SuggestOptions) (*Suggestion, error) {
	if o.WorkStart < 0 || o.WorkEnd > minutesPerDay || o.WorkEnd-o.WorkStart < MinBlockMinutes {
		return nil, invalidError("working hours must leave room for at least one block")
	}
	plan, err := s.Plan(ctx, o.Date)
	if err != nil {
		return nil, err
	}
	in := DayInput{Date: o.Date, WorkStart: o.WorkStart, WorkEnd: o.WorkEnd}
	if o.Date == s.lists.ToDay() {
		in.NowMin = o.NowMin
	}
	in.Busy = append(in.Busy, plan.Busy...)
	placed := map[int64]int{}
	for _, b := range plan.Blocks {
		placed[b.TaskID] += b.DurationMin
		in.Busy = append(in.Busy, Interval{StartMin: b.StartMin, EndMin: b.EndMin()})
	}
	for _, t := range plan.Tasks {
		need := t.EstimateMinutes - placed[t.ID]
		if t.EstimateMinutes == 0 && placed[t.ID] == 0 {
			need = DefaultBlockMinutes
		}
		if need > 0 {
			in.Tasks = append(in.Tasks, PlanTask{Task: t, Minutes: need})
		}
	}

	if !o.UseAI {
		props, unplaced := autoSchedule(in)
		return &Suggestion{Source: "heuristic", Proposals: props, Unplaced: unplaced, Summary: heuristicSummary(props, unplaced)}, nil
	}
	if s.advisor == nil {
		return nil, invalidError("no AI is set up")
	}
	if len(in.Tasks) == 0 {
		return &Suggestion{Source: "ai", Summary: "Everything already has time set aside."}, nil
	}
	summary, raw, err := s.advisor.Advise(ctx, in)
	if err != nil {
		return nil, err
	}
	props, dropped := validateProposals(in, raw)
	sg := &Suggestion{Source: "ai", Summary: summary, Proposals: props, Dropped: dropped}
	got := map[int64]bool{}
	for _, p := range props {
		got[p.TaskID] = true
	}
	for _, t := range in.Tasks {
		if !got[t.Task.ID] {
			sg.Unplaced = append(sg.Unplaced, t.Task.ID)
		}
	}
	return sg, nil
}

func heuristicSummary(props []Proposal, unplaced []int64) string {
	switch {
	case len(props) == 0 && len(unplaced) == 0:
		return "Everything already has time set aside."
	case len(props) == 0:
		return "There is no free time left in your working hours."
	case len(unplaced) > 0:
		return fmt.Sprintf("Placed %d; %d did not fit in your working hours.", len(props), len(unplaced))
	}
	return fmt.Sprintf("Placed %d in your free time, most urgent first.", len(props))
}

// ---- the built-in scheduler

type span struct{ start, end int }

// freeSpans is [from, WorkEnd) minus the busy intervals, trimmed to the grid.
func freeSpans(in DayInput, grid int) []span {
	from := in.WorkStart
	if in.NowMin > from {
		from = in.NowMin
	}
	from = ceilTo(from, grid)
	free := []span{{from, floorTo(in.WorkEnd, grid)}}
	busy := append([]Interval(nil), in.Busy...)
	sort.Slice(busy, func(i, j int) bool { return busy[i].StartMin < busy[j].StartMin })
	for _, b := range busy {
		var next []span
		for _, f := range free {
			if b.EndMin <= f.start || b.StartMin >= f.end {
				next = append(next, f)
				continue
			}
			if b.StartMin > f.start {
				next = append(next, span{f.start, floorTo(b.StartMin, grid)})
			}
			if b.EndMin < f.end {
				next = append(next, span{ceilTo(b.EndMin, grid), f.end})
			}
		}
		free = next
	}
	var out []span
	for _, f := range free {
		if f.end > f.start {
			out = append(out, f)
		}
	}
	return out
}

func ceilTo(n, g int) int  { return (n + g - 1) / g * g }
func floorTo(n, g int) int { return n / g * g }

// urgency orders tasks: due today or overdue first, then priority, then the
// nearest deadline; ties keep list order.
func urgency(t models.Task, date string) (rank int, reason string) {
	switch {
	case t.Deadline != "" && t.Deadline < date:
		return 0, "Overdue"
	case t.Deadline == date:
		return 0, "Due today"
	case t.Priority >= models.PriorityHigh:
		return 1, "High priority"
	}
	return 1, ""
}

// autoSchedule places each task whole in the earliest free span that fits it.
func autoSchedule(in DayInput) (props []Proposal, unplaced []int64) {
	tasks := append([]PlanTask(nil), in.Tasks...)
	sort.SliceStable(tasks, func(i, j int) bool {
		ri, _ := urgency(tasks[i].Task, in.Date)
		rj, _ := urgency(tasks[j].Task, in.Date)
		if ri != rj {
			return ri < rj
		}
		if tasks[i].Task.Priority != tasks[j].Task.Priority {
			return tasks[i].Task.Priority > tasks[j].Task.Priority
		}
		di, dj := tasks[i].Task.Deadline, tasks[j].Task.Deadline
		if di != dj {
			return di != "" && (dj == "" || di < dj)
		}
		return false
	})
	free := freeSpans(in, suggestSnap)
	for _, t := range tasks {
		need := ceilTo(t.Minutes, suggestSnap)
		placed := false
		for i, f := range free {
			if f.end-f.start < need {
				continue
			}
			_, reason := urgency(t.Task, in.Date)
			props = append(props, Proposal{TaskID: t.Task.ID, StartMin: f.start, DurationMin: t.Minutes, Reason: reason})
			free[i].start += need
			placed = true
			break
		}
		if !placed {
			unplaced = append(unplaced, t.Task.ID)
		}
	}
	sort.Slice(props, func(i, j int) bool { return props[i].StartMin < props[j].StartMin })
	return props, unplaced
}

// ---- validating an Advisor's answer

// validateProposals keeps only proposals that are safe to apply: a known task,
// within working hours and not in the past, not overlapping busy time or each
// other, and no more minutes per task than it still needs.
func validateProposals(in DayInput, raw []Proposal) (kept []Proposal, dropped []string) {
	need := map[int64]PlanTask{}
	for _, t := range in.Tasks {
		need[t.Task.ID] = t
	}
	used := map[int64]int{}
	occupied := append([]Interval(nil), in.Busy...)
	from := in.WorkStart
	if in.NowMin > from {
		from = in.NowMin
	}
	for _, p := range raw {
		t, ok := need[p.TaskID]
		if !ok {
			dropped = append(dropped, fmt.Sprintf("task %d is not one of today's open tasks", p.TaskID))
			continue
		}
		label := "“" + t.Task.Title + "”"
		end := p.StartMin + p.DurationMin
		switch {
		case p.DurationMin < MinBlockMinutes:
			dropped = append(dropped, label+": too short")
		case p.StartMin%aiSnap != 0 || p.DurationMin%aiSnap != 0:
			dropped = append(dropped, label+": not on a 5-minute boundary")
		case p.StartMin < from || end > in.WorkEnd:
			dropped = append(dropped, label+": outside your working hours")
		case used[p.TaskID]+p.DurationMin > t.Minutes:
			dropped = append(dropped, label+": more time than it needs")
		default:
			clash := false
			for _, o := range occupied {
				if p.StartMin < o.EndMin && o.StartMin < end {
					clash = true
					break
				}
			}
			if clash {
				dropped = append(dropped, label+": overlaps something already booked")
				continue
			}
			used[p.TaskID] += p.DurationMin
			occupied = append(occupied, Interval{StartMin: p.StartMin, EndMin: end})
			kept = append(kept, p)
		}
	}
	sort.Slice(kept, func(i, j int) bool { return kept[i].StartMin < kept[j].StartMin })
	return kept, dropped
}

// Apply places the proposed blocks together: either all of them are added or,
// if any no longer fits (the day changed since the suggestion), none are.
func (s *PlanningService) Apply(date string, props []Proposal) ([]models.TimeBlock, error) {
	if err := checkDate(date); err != nil {
		return nil, err
	}
	var out []models.TimeBlock
	err := s.store.WithTx(func(tx storage.Store) error {
		out = nil
		for _, p := range props {
			if err := checkSlot(date, p.StartMin, p.DurationMin); err != nil {
				return err
			}
			t, err := tx.GetTask(p.TaskID)
			if errors.Is(err, storage.ErrNotFound) {
				return &NotFoundError{Entity: "task", ID: p.TaskID}
			}
			if err != nil {
				return err
			}
			if t.Status != models.StatusOpen {
				return invalidError("“" + t.Title + "” is no longer open")
			}
			b := &models.TimeBlock{TaskID: p.TaskID, Date: date, StartMin: p.StartMin, DurationMin: p.DurationMin}
			if err := checkFree(tx, b); err != nil {
				return invalidError("the day changed since this was suggested; plan it again")
			}
			if err := tx.AddTimeBlock(b); err != nil {
				return err
			}
			out = append(out, *b)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.changed()
	return out, nil
}
