package appapi

import (
	"context"
	"fmt"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/ai"
	"github.com/yalcinumutt/zenith-cli/internal/calendar"
	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

// ---- adapters from the outside world to the core's small interfaces

// calendarBusy serves ICS events to the planner as busy intervals.
type calendarBusy struct {
	feed *calendar.Feed
	loc  *time.Location
}

func (c calendarBusy) Busy(ctx context.Context, date string) ([]core.Interval, error) {
	events, err := c.feed.Busy(ctx, date, c.loc)
	out := make([]core.Interval, 0, len(events))
	for _, e := range events {
		out = append(out, core.Interval{Title: e.Title, StartMin: e.StartMin, EndMin: e.EndMin})
	}
	return out, err
}

// aiAdvisor asks the configured model to place a day's tasks. Only scheduling
// facts leave the process: titles of the tasks to place, minutes, priority and
// deadlines, and anonymous busy spans (never event titles or notes).
type aiAdvisor struct{ s *Service }

func (a aiAdvisor) Advise(ctx context.Context, in core.DayInput) (string, []core.Proposal, error) {
	cfg := a.s.aiConfig()
	if !cfg.Ready() {
		return "", nil, errAINotReady
	}
	req := ai.PlanRequest{Date: in.Date, WorkStart: in.WorkStart, WorkEnd: in.WorkEnd, EarliestM: max(in.WorkStart, in.NowMin)}
	for _, t := range in.Tasks {
		req.Tasks = append(req.Tasks, ai.PlanTask{ID: t.Task.ID, Title: t.Task.Title, Minutes: t.Minutes, Priority: int(t.Task.Priority), Deadline: t.Task.Deadline})
	}
	for _, b := range in.Busy {
		req.Busy = append(req.Busy, ai.Span{Start: b.StartMin, End: b.EndMin})
	}
	summary, placed, err := cfg.PlanDay(ctx, req)
	if err != nil {
		return "", nil, err
	}
	props := make([]core.Proposal, 0, len(placed))
	for _, p := range placed {
		props = append(props, core.Proposal{TaskID: p.TaskID, StartMin: p.Start, DurationMin: p.Duration, Reason: p.Reason})
	}
	return summary, props, nil
}

type aiNotReady struct{}

func (aiNotReady) Error() string { return "set up an AI provider in the planner settings first" }

var errAINotReady error = aiNotReady{}

// ---- daily planning

func (s *Service) nowMin() int {
	n := s.clock.Now()
	return n.Hour()*60 + n.Minute()
}

// Plan returns a day's tasks, time blocks and calendar events. An empty date means today.
func (s *Service) Plan(date string) (PlanDTO, error) {
	if date == "" {
		date = s.today()
	}
	p, err := s.core.Planning.Plan(context.Background(), date)
	if err != nil {
		return PlanDTO{}, err
	}
	tasks, err := s.toDTOs(p.Tasks)
	if err != nil {
		return PlanDTO{}, err
	}
	// A block can belong to a task that is not in the day's list (completed
	// since, or scheduled elsewhere); resolve its title individually.
	listed := map[int64]models.Task{}
	for _, t := range p.Tasks {
		listed[t.ID] = t
	}
	blocks := make([]BlockDTO, 0, len(p.Blocks))
	for _, b := range p.Blocks {
		t, ok := listed[b.TaskID]
		if !ok {
			full, err := s.core.Tasks.Get(b.TaskID)
			if err != nil {
				continue // task vanished between the two reads
			}
			t = *full
		}
		blocks = append(blocks, BlockDTO{
			ID: b.ID, UUID: b.UUID, TaskID: b.TaskID, TaskTitle: t.Title, Done: t.Status != models.StatusOpen,
			Date: b.Date, Start: b.StartMin, Duration: b.DurationMin,
		})
	}
	busy := make([]BusyDTO, 0, len(p.Busy))
	for _, b := range p.Busy {
		busy = append(busy, BusyDTO{Title: b.Title, Start: b.StartMin, End: b.EndMin})
	}
	st := s.plan.load()
	cfg := s.aiConfig()
	dto := PlanDTO{
		Date: date, Tasks: tasks, Blocks: blocks, Busy: busy, CalendarConfigured: s.feed.Configured(),
		WorkStart: st.WorkStart, WorkEnd: st.WorkEnd, AIReady: cfg.Ready(), AICloud: cfg.Ready() && cfg.IsCloud(),
	}
	if p.BusyErr != nil {
		dto.CalendarError = p.BusyErr.Error()
	}
	return dto, nil
}

func (s *Service) AddBlock(taskID int64, date string, start, duration int) (BlockDTO, error) {
	b, err := s.core.Planning.AddBlock(taskID, date, start, duration)
	if err != nil {
		return BlockDTO{}, err
	}
	t, err := s.core.Tasks.Get(taskID)
	if err != nil {
		return BlockDTO{}, err
	}
	return BlockDTO{ID: b.ID, UUID: b.UUID, TaskID: taskID, TaskTitle: t.Title, Date: b.Date, Start: b.StartMin, Duration: b.DurationMin}, nil
}

func (s *Service) MoveBlock(id int64, date string, start int) error {
	return s.core.Planning.Move(id, date, start)
}

func (s *Service) ResizeBlock(id int64, duration int) error {
	return s.core.Planning.Resize(id, duration)
}
func (s *Service) RemoveBlock(id int64) error { return s.core.Planning.Remove(id) }

// SuggestPlan proposes blocks for a day without saving anything. With useAI it
// asks the configured model; otherwise it uses the built-in scheduler.
func (s *Service) SuggestPlan(date string, useAI bool) (SuggestionDTO, error) {
	if date == "" {
		date = s.today()
	}
	st := s.plan.load()
	sg, err := s.core.Planning.Suggest(context.Background(), core.SuggestOptions{
		Date: date, WorkStart: st.WorkStart, WorkEnd: st.WorkEnd, UseAI: useAI, NowMin: s.nowMin(),
	})
	if err != nil {
		return SuggestionDTO{}, err
	}
	titles := map[int64]string{}
	titleOf := func(id int64) string {
		if t, ok := titles[id]; ok {
			return t
		}
		if t, err := s.core.Tasks.Get(id); err == nil {
			titles[id] = t.Title
		}
		return titles[id]
	}
	out := SuggestionDTO{Source: sg.Source, Summary: sg.Summary, Proposals: []ProposalDTO{}, Unplaced: []string{}, Dropped: sg.Dropped}
	if out.Dropped == nil {
		out.Dropped = []string{}
	}
	for _, p := range sg.Proposals {
		out.Proposals = append(out.Proposals, ProposalDTO{TaskID: p.TaskID, TaskTitle: titleOf(p.TaskID), Start: p.StartMin, Duration: p.DurationMin, Reason: p.Reason})
	}
	for _, id := range sg.Unplaced {
		out.Unplaced = append(out.Unplaced, titleOf(id))
	}
	return out, nil
}

// ApplySuggestion saves a reviewed suggestion as time blocks, all or nothing.
func (s *Service) ApplySuggestion(date string, proposals []ProposalDTO) error {
	if date == "" {
		date = s.today()
	}
	props := make([]core.Proposal, 0, len(proposals))
	for _, p := range proposals {
		props = append(props, core.Proposal{TaskID: p.TaskID, StartMin: p.Start, DurationMin: p.Duration})
	}
	_, err := s.core.Planning.Apply(date, props)
	return err
}

// TestCalendar fetches the saved calendar link now and says whether it works.
func (s *Service) TestCalendar() CalendarTestDTO {
	if !s.feed.Configured() {
		return CalendarTestDTO{Message: "No calendar link is saved."}
	}
	if err := s.feed.Refresh(context.Background()); err != nil {
		return CalendarTestDTO{Message: err.Error()}
	}
	events, _ := s.feed.Busy(context.Background(), s.today(), s.clock.Now().Location())
	return CalendarTestDTO{OK: true, Message: pluralEvents(len(events)) + " today."}
}

// TestCalendarURL tries a link that may not be saved yet, without changing any setting.
func (s *Service) TestCalendarURL(raw string) CalendarTestDTO {
	u, err := calendar.Normalize(raw)
	if err != nil {
		return CalendarTestDTO{Message: err.Error()}
	}
	if u == "" {
		return CalendarTestDTO{Message: "Paste a calendar link first."}
	}
	f := calendar.NewFeed(nil)
	f.SetLocation(s.clock.Now().Location())
	_ = f.SetURL(u)
	if err := f.Refresh(context.Background()); err != nil {
		return CalendarTestDTO{Message: err.Error()}
	}
	events, _ := f.Busy(context.Background(), s.today(), s.clock.Now().Location())
	return CalendarTestDTO{OK: true, Message: pluralEvents(len(events)) + " today."}
}

func pluralEvents(n int) string {
	if n == 1 {
		return "Connected: 1 event"
	}
	return fmt.Sprintf("Connected: %d events", n)
}
