package calendar

import (
	"strconv"
	"strings"
	"time"
)

type rule struct {
	freq       string
	interval   int
	count      int
	until      time.Time
	byDay      []time.Weekday
	byMonthDay []int
}

func parseRule(s string, local *time.Location) (rule, bool) {
	r := rule{interval: 1}
	for _, part := range strings.Split(s, ";") {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		switch strings.ToUpper(k) {
		case "FREQ":
			r.freq = strings.ToUpper(v)
		case "INTERVAL":
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				r.interval = n
			}
		case "COUNT":
			r.count, _ = strconv.Atoi(v)
		case "UNTIL":
			if t, _ := parseTime(v, nil, local); !t.IsZero() {
				r.until = t
				if len(v) == 8 { // a date: through the end of that day
					r.until = t.AddDate(0, 0, 1).Add(-time.Second)
				}
			}
		case "BYDAY":
			for _, d := range strings.Split(v, ",") {
				if len(d) >= 2 {
					if wd, ok := weekdays[strings.ToUpper(d[len(d)-2:])]; ok {
						r.byDay = append(r.byDay, wd)
					}
				}
			}
		case "BYMONTHDAY":
			for _, d := range strings.Split(v, ",") {
				if n, err := strconv.Atoi(d); err == nil {
					r.byMonthDay = append(r.byMonthDay, n)
				}
			}
		}
	}
	switch r.freq {
	case "DAILY", "WEEKLY", "MONTHLY", "YEARLY":
		return r, true
	}
	return r, false
}

var weekdays = map[string]time.Weekday{"SU": time.Sunday, "MO": time.Monday, "TU": time.Tuesday, "WE": time.Wednesday, "TH": time.Thursday, "FR": time.Friday, "SA": time.Saturday}

// occurrences returns the start times of an event's recurrences that begin in
// [from, to), expanded in the event's own start zone so wall-clock times hold
// across daylight-saving changes. Supported: DAILY, WEEKLY (BYDAY), MONTHLY
// (BYMONTHDAY or the start's day) and YEARLY, with INTERVAL, COUNT and UNTIL.
func occurrences(e rawEvent, from, to time.Time, local *time.Location) []time.Time {
	r, ok := parseRule(e.rrule, local)
	if !ok {
		return nil
	}
	loc := e.start.Location()
	start := e.start
	var out []time.Time
	n := 0 // occurrences generated so far, for COUNT

	emit := func(t time.Time) bool { // false stops the expansion
		if !r.until.IsZero() && t.After(r.until) {
			return false
		}
		if t.Before(start) {
			return true
		}
		n++
		if r.count > 0 && n > r.count {
			return false
		}
		if !t.Before(to) {
			return false
		}
		if !t.Before(from) {
			out = append(out, t)
		}
		return true
	}
	at := func(y int, m time.Month, d int) time.Time {
		return time.Date(y, m, d, start.Hour(), start.Minute(), start.Second(), 0, loc)
	}

	const guard = 5000 // never loop forever on a degenerate rule
	switch r.freq {
	case "DAILY":
		for i := 0; i < guard; i++ {
			if !emit(at(start.Year(), start.Month(), start.Day()+i*r.interval)) {
				break
			}
		}
	case "WEEKLY":
		days := r.byDay
		if len(days) == 0 {
			days = []time.Weekday{start.Weekday()}
		}
		weekStart := start.AddDate(0, 0, -int(start.Weekday())) // Sunday of the first week
		for w := 0; w < guard; w++ {
			base := weekStart.AddDate(0, 0, 7*w*r.interval)
			stop := false
			for wd := time.Sunday; wd <= time.Saturday; wd++ {
				if !contains(days, wd) {
					continue
				}
				if !emit(at(base.Year(), base.Month(), base.Day()+int(wd))) {
					stop = true
					break
				}
			}
			if stop {
				break
			}
		}
	case "MONTHLY":
		mdays := r.byMonthDay
		if len(mdays) == 0 {
			mdays = []int{start.Day()}
		}
		for m := 0; m < guard; m++ {
			first := time.Date(start.Year(), start.Month()+time.Month(m*r.interval), 1, 0, 0, 0, 0, loc)
			stop := false
			for _, d := range sortedInts(mdays) {
				if d < 1 || d > daysIn(first) {
					continue
				}
				if !emit(at(first.Year(), first.Month(), d)) {
					stop = true
					break
				}
			}
			if stop {
				break
			}
		}
	case "YEARLY":
		for y := 0; y < guard; y++ {
			t := at(start.Year()+y*r.interval, start.Month(), start.Day())
			if t.Month() != start.Month() { // Feb 29 in a non-leap year
				continue
			}
			if !emit(t) {
				break
			}
		}
	}
	return out
}

func contains(ds []time.Weekday, d time.Weekday) bool {
	for _, x := range ds {
		if x == d {
			return true
		}
	}
	return false
}

func daysIn(firstOfMonth time.Time) int {
	return firstOfMonth.AddDate(0, 1, -1).Day()
}

func sortedInts(in []int) []int {
	out := append([]int(nil), in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j] < out[j-1]; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
