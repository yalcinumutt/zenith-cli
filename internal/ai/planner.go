package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// PlanTask is one task to place. Only what a scheduler needs is sent: no notes,
// no project names, no tags.
type PlanTask struct {
	ID       int64
	Title    string
	Minutes  int    // minutes still to place
	Priority int    // 0 low .. 3 critical
	Deadline string // YYYY-MM-DD or ""
}

// Span is a busy stretch of the day in minutes after midnight. It carries no title.
type Span struct{ Start, End int }

// PlanRequest describes one day to the model.
type PlanRequest struct {
	Date      string
	WorkStart int
	WorkEnd   int
	EarliestM int // earliest usable minute (now, for today)
	Tasks     []PlanTask
	Busy      []Span
}

// Placement is one block the model chose.
type Placement struct {
	TaskID   int64
	Start    int
	Duration int
	Reason   string
}

const plannerSystem = `You are a scheduling assistant inside a task app. You place tasks into free time on one day.
Reply with a single JSON object and nothing else, in this shape:
{"summary": "one short sentence", "blocks": [{"task_id": 1, "start": "09:00", "duration": 45, "reason": "short"}]}
Rules: use only the given task ids; "start" is 24-hour HH:MM on a 5-minute boundary; "duration" is in minutes and the total per task must not exceed its "minutes"; blocks must lie inside the working hours and from the earliest start, must not overlap each other or any busy span; put urgent and high-priority work earlier, and demanding tasks in the first half of the day when possible; leave a task out if it does not fit.`

func clock(m int) string { return fmt.Sprintf("%02d:%02d", m/60, m%60) }

func buildPrompt(r PlanRequest) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Date: %s\nWorking hours: %s-%s\nEarliest start: %s\n\nBusy spans (do not use):\n", r.Date, clock(r.WorkStart), clock(r.WorkEnd), clock(r.EarliestM))
	if len(r.Busy) == 0 {
		b.WriteString("- none\n")
	}
	for _, s := range r.Busy {
		fmt.Fprintf(&b, "- %s-%s\n", clock(s.Start), clock(s.End))
	}
	b.WriteString("\nTasks to place:\n")
	for _, t := range r.Tasks {
		dl := ""
		if t.Deadline != "" {
			dl = ", deadline " + t.Deadline
		}
		fmt.Fprintf(&b, "- id %d: %q, %d minutes, priority %d of 3%s\n", t.ID, t.Title, t.Minutes, t.Priority, dl)
	}
	return b.String()
}

// PlanDay asks the model to place the tasks and parses its JSON answer. The
// answer is not trusted: callers must validate every placement.
func (c ProviderConfig) PlanDay(ctx context.Context, r PlanRequest) (summary string, out []Placement, err error) {
	text, err := c.Complete(ctx, plannerSystem, buildPrompt(r))
	if err != nil {
		return "", nil, err
	}
	return parsePlan(text)
}

func parsePlan(text string) (string, []Placement, error) {
	obj := extractJSON(text)
	if obj == "" {
		return "", nil, errors.New("the AI did not answer with a plan; try again")
	}
	var raw struct {
		Summary string `json:"summary"`
		Blocks  []struct {
			TaskID   int64  `json:"task_id"`
			Start    string `json:"start"`
			Duration int    `json:"duration"`
			Reason   string `json:"reason"`
		} `json:"blocks"`
	}
	if err := json.Unmarshal([]byte(obj), &raw); err != nil {
		return "", nil, errors.New("the AI answered with a malformed plan; try again")
	}
	var out []Placement
	for _, b := range raw.Blocks {
		m, ok := parseClock(b.Start)
		if !ok {
			continue // an unreadable time is just a dropped block
		}
		out = append(out, Placement{TaskID: b.TaskID, Start: m, Duration: b.Duration, Reason: strings.TrimSpace(b.Reason)})
	}
	return strings.TrimSpace(raw.Summary), out, nil
}

func parseClock(s string) (int, bool) {
	h, m, ok := strings.Cut(strings.TrimSpace(s), ":")
	if !ok {
		return 0, false
	}
	hi, err1 := strconv.Atoi(h)
	mi, err2 := strconv.Atoi(m)
	if err1 != nil || err2 != nil || hi < 0 || hi > 23 || mi < 0 || mi > 59 {
		return 0, false
	}
	return hi*60 + mi, true
}

// extractJSON returns the first balanced {...} object in text, tolerating
// markdown fences and chatter around it.
func extractJSON(text string) string {
	start := strings.IndexByte(text, '{')
	if start < 0 {
		return ""
	}
	depth, inStr, esc := 0, false, false
	for i := start; i < len(text); i++ {
		ch := text[i]
		switch {
		case esc:
			esc = false
		case inStr && ch == '\\':
			esc = true
		case ch == '"':
			inStr = !inStr
		case inStr:
		case ch == '{':
			depth++
		case ch == '}':
			depth--
			if depth == 0 {
				return text[start : i+1]
			}
		}
	}
	return ""
}
