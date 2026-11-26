// Package calendar reads a read-only iCalendar (.ics) feed and answers one
// question: which parts of a day are busy. It knows nothing of tasks or the
// core; the planner is handed its result through a small interface.
package calendar

import (
	"sort"
	"strconv"
	"strings"
	"time"
)

// Event is one busy stretch on one day.
type Event struct {
	Title    string
	StartMin int // minutes after local midnight
	EndMin   int // exclusive; clipped to the day
}

// rawEvent is a parsed VEVENT before recurrence expansion.
type rawEvent struct {
	title      string
	start, end time.Time
	allDay     bool
	rrule      string
	exdates    map[string]bool // local YYYY-MM-DD of excluded occurrences
	recurID    string          // RECURRENCE-ID day: this event replaces that occurrence
	uid        string
	skip       bool // cancelled or transparent
}

// Calendar is a parsed feed.
type Calendar struct{ events []rawEvent }

// Parse reads iCalendar text. Unknown or malformed events are skipped, not fatal:
// a feed with one odd event should still show the rest of the day.
func Parse(text string, local *time.Location) *Calendar {
	if local == nil {
		local = time.Local
	}
	cal := &Calendar{}
	var cur *rawEvent
	for _, line := range unfold(text) {
		name, params, value := splitLine(line)
		switch {
		case name == "BEGIN" && value == "VEVENT":
			cur = &rawEvent{exdates: map[string]bool{}}
		case name == "END" && value == "VEVENT":
			if cur != nil && !cur.skip && !cur.start.IsZero() {
				if cur.end.IsZero() || cur.end.Before(cur.start) {
					cur.end = cur.start
				}
				cal.events = append(cal.events, *cur)
			}
			cur = nil
		case cur == nil:
		case name == "SUMMARY":
			cur.title = unescape(value)
		case name == "UID":
			cur.uid = value
		case name == "STATUS":
			cur.skip = cur.skip || strings.EqualFold(value, "CANCELLED")
		case name == "TRANSP":
			cur.skip = cur.skip || strings.EqualFold(value, "TRANSPARENT")
		case name == "DTSTART":
			cur.start, cur.allDay = parseTime(value, params, local)
		case name == "DTEND":
			cur.end, _ = parseTime(value, params, local)
		case name == "DURATION":
			if d, ok := parseDuration(value); ok && !cur.start.IsZero() {
				cur.end = cur.start.Add(d)
			}
		case name == "RRULE":
			cur.rrule = value
		case name == "EXDATE":
			for _, v := range strings.Split(value, ",") {
				if t, _ := parseTime(v, params, local); !t.IsZero() {
					cur.exdates[t.In(local).Format("2006-01-02")] = true
				}
			}
		case name == "RECURRENCE-ID":
			if t, _ := parseTime(value, params, local); !t.IsZero() {
				cur.recurID = t.In(local).Format("2006-01-02")
			}
		}
	}
	return cal
}

// On returns the busy events on one local day (YYYY-MM-DD), by start time.
// All-day events are left out: they mark a day, not a stretch of it.
func (c *Calendar) On(day string, local *time.Location) []Event {
	if local == nil {
		local = time.Local
	}
	d0, err := time.ParseInLocation("2006-01-02", day, local)
	if err != nil {
		return nil
	}
	d1 := d0.AddDate(0, 0, 1)

	// Instances replaced by a RECURRENCE-ID override are not generated again.
	overridden := map[string]bool{}
	for _, e := range c.events {
		if e.recurID != "" {
			overridden[e.uid+"|"+e.recurID] = true
		}
	}

	var out []Event
	add := func(title string, s, e time.Time) {
		if !s.Before(d1) || !e.After(d0) {
			return
		}
		sm, em := 0, 24*60
		if s.After(d0) {
			sm = int(s.Sub(d0) / time.Minute)
		}
		if e.Before(d1) {
			em = int(e.Sub(d0) / time.Minute)
		}
		if em <= sm {
			return // zero-length
		}
		out = append(out, Event{Title: title, StartMin: sm, EndMin: em})
	}
	for _, e := range c.events {
		if e.allDay {
			continue
		}
		if e.rrule == "" {
			add(e.title, e.start, e.end)
			continue
		}
		dur := e.end.Sub(e.start)
		for _, s := range occurrences(e, d0.AddDate(0, 0, -2), d1, local) {
			key := s.In(local).Format("2006-01-02")
			if e.exdates[key] || overridden[e.uid+"|"+key] {
				continue
			}
			add(e.title, s, s.Add(dur))
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].StartMin < out[j].StartMin })
	return out
}

// ---- parsing helpers

// unfold joins continuation lines (a line starting with space or tab continues the previous).
func unfold(text string) []string {
	text = strings.ReplaceAll(strings.ReplaceAll(text, "\r\n", "\n"), "\r", "\n")
	var lines []string
	for _, l := range strings.Split(text, "\n") {
		if (strings.HasPrefix(l, " ") || strings.HasPrefix(l, "\t")) && len(lines) > 0 {
			lines[len(lines)-1] += l[1:]
			continue
		}
		if l != "" {
			lines = append(lines, l)
		}
	}
	return lines
}

// splitLine splits "NAME;P=V;Q=W:value" at the first colon outside quotes.
func splitLine(line string) (name string, params map[string]string, value string) {
	inQuote, cut := false, -1
	for i, r := range line {
		if r == '"' {
			inQuote = !inQuote
		}
		if r == ':' && !inQuote {
			cut = i
			break
		}
	}
	if cut < 0 {
		return "", nil, ""
	}
	head, value := line[:cut], line[cut+1:]
	parts := strings.Split(head, ";")
	name = strings.ToUpper(parts[0])
	params = map[string]string{}
	for _, p := range parts[1:] {
		if k, v, ok := strings.Cut(p, "="); ok {
			params[strings.ToUpper(k)] = strings.Trim(v, `"`)
		}
	}
	return name, params, value
}

func unescape(s string) string {
	r := strings.NewReplacer(`\n`, " ", `\N`, " ", `\,`, ",", `\;`, ";", `\\`, `\`)
	return r.Replace(s)
}

// parseTime reads a DATE, floating DATE-TIME, UTC DATE-TIME or TZID DATE-TIME.
// Floating and unknown-zone times are taken in the local zone.
func parseTime(value string, params map[string]string, local *time.Location) (t time.Time, allDay bool) {
	value = strings.TrimSpace(value)
	if params["VALUE"] == "DATE" || len(value) == 8 {
		t, err := time.ParseInLocation("20060102", value, local)
		if err != nil {
			return time.Time{}, false
		}
		return t, true
	}
	if strings.HasSuffix(value, "Z") {
		t, err := time.Parse("20060102T150405Z", value)
		if err != nil {
			return time.Time{}, false
		}
		return t, false
	}
	loc := local
	if tz := params["TZID"]; tz != "" {
		if l, err := time.LoadLocation(tz); err == nil {
			loc = l
		}
	}
	t, err := time.ParseInLocation("20060102T150405", value, loc)
	if err != nil {
		return time.Time{}, false
	}
	return t, false
}

// parseDuration reads an RFC 5545 duration such as PT1H30M, P1D or P1W.
func parseDuration(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	neg := strings.HasPrefix(s, "-")
	s = strings.TrimLeft(s, "+-")
	if !strings.HasPrefix(s, "P") {
		return 0, false
	}
	s = s[1:]
	var total time.Duration
	inTime := false
	num := ""
	for _, r := range s {
		switch {
		case r == 'T':
			inTime = true
		case r >= '0' && r <= '9':
			num += string(r)
		default:
			n, err := strconv.Atoi(num)
			if err != nil {
				return 0, false
			}
			num = ""
			switch {
			case r == 'W':
				total += time.Duration(n) * 7 * 24 * time.Hour
			case r == 'D':
				total += time.Duration(n) * 24 * time.Hour
			case r == 'H' && inTime:
				total += time.Duration(n) * time.Hour
			case r == 'M' && inTime:
				total += time.Duration(n) * time.Minute
			case r == 'S' && inTime:
				total += time.Duration(n) * time.Second
			default:
				return 0, false
			}
		}
	}
	if neg {
		total = -total
	}
	return total, true
}
