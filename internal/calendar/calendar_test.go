package calendar

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

var berlin = mustLoc("Europe/Berlin")

func mustLoc(n string) *time.Location {
	l, err := time.LoadLocation(n)
	if err != nil {
		panic(err)
	}
	return l
}

func wrap(events ...string) string {
	s := "BEGIN:VCALENDAR\r\nVERSION:2.0\r\n"
	for _, e := range events {
		s += "BEGIN:VEVENT\r\n" + e + "END:VEVENT\r\n"
	}
	return s + "END:VCALENDAR\r\n"
}

func spans(es []Event) [][3]any {
	var out [][3]any
	for _, e := range es {
		out = append(out, [3]any{e.Title, e.StartMin, e.EndMin})
	}
	return out
}

func TestSingleEventsAndZones(t *testing.T) {
	cal := Parse(wrap(
		"SUMMARY:Standup\r\nDTSTART;TZID=Europe/Berlin:20260301T093000\r\nDTEND;TZID=Europe/Berlin:20260301T100000\r\n",
		"SUMMARY:UTC call\r\nDTSTART:20260301T130000Z\r\nDTEND:20260301T140000Z\r\n", // 14:00-15:00 in Berlin (UTC+1)
		"SUMMARY:Floating\r\nDTSTART:20260301T170000\r\nDURATION:PT90M\r\n",
		"SUMMARY:Holiday\r\nDTSTART;VALUE=DATE:20260301\r\n", // all-day: not busy time
		"SUMMARY:Cancelled\r\nSTATUS:CANCELLED\r\nDTSTART:20260301T110000\r\nDTEND:20260301T120000\r\n",
		"SUMMARY:Free\r\nTRANSP:TRANSPARENT\r\nDTSTART:20260301T110000\r\nDTEND:20260301T120000\r\n",
		"SUMMARY:Other day\r\nDTSTART:20260302T090000\r\nDTEND:20260302T100000\r\n",
	), berlin)
	got := spans(cal.On("2026-03-01", berlin))
	want := [][3]any{{"Standup", 570, 600}, {"UTC call", 840, 900}, {"Floating", 1020, 1110}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestEventSpanningMidnightIsClipped(t *testing.T) {
	cal := Parse(wrap("SUMMARY:Night\r\nDTSTART:20260301T230000\r\nDTEND:20260302T010000\r\n"), berlin)
	if got := spans(cal.On("2026-03-01", berlin)); !reflect.DeepEqual(got, [][3]any{{"Night", 1380, 1440}}) {
		t.Fatalf("day 1: %v", got)
	}
	if got := spans(cal.On("2026-03-02", berlin)); !reflect.DeepEqual(got, [][3]any{{"Night", 0, 60}}) {
		t.Fatalf("day 2: %v", got)
	}
}

func TestFoldedLinesAndEscapes(t *testing.T) {
	cal := Parse("BEGIN:VEVENT\r\nSUMMARY:Planning\\, part\r\n  two\r\nDTSTART:20260301T090000\r\nDTEND:20260301T100000\r\nEND:VEVENT\r\n", berlin)
	got := cal.On("2026-03-01", berlin)
	if len(got) != 1 || got[0].Title != "Planning, part two" {
		t.Fatalf("got %+v", got)
	}
}

func TestRecurrence(t *testing.T) {
	cal := Parse(wrap(
		// Mon/Wed/Fri 09:00-09:30 starting Mon 2026-03-02, ending after 4 occurrences.
		"UID:a\r\nSUMMARY:MWF\r\nDTSTART;TZID=Europe/Berlin:20260302T090000\r\nDTEND;TZID=Europe/Berlin:20260302T093000\r\nRRULE:FREQ=WEEKLY;BYDAY=MO,WE,FR;COUNT=4\r\n",
		// Every second day, minus one excluded date.
		"UID:b\r\nSUMMARY:Alt\r\nDTSTART:20260301T200000\r\nDTEND:20260301T210000\r\nRRULE:FREQ=DAILY;INTERVAL=2\r\nEXDATE:20260305T200000\r\n",
		// Monthly on the 15th until a date.
		"UID:c\r\nSUMMARY:Rent\r\nDTSTART:20260115T120000\r\nDTEND:20260115T123000\r\nRRULE:FREQ=MONTHLY;BYMONTHDAY=15;UNTIL=20260331T000000Z\r\n",
	), berlin)
	has := func(day, title string) bool {
		for _, e := range cal.On(day, berlin) {
			if e.Title == title {
				return true
			}
		}
		return false
	}
	// MWF: Mar 2, 4, 6, 9 then COUNT ends the series.
	for day, want := range map[string]bool{"2026-03-02": true, "2026-03-03": false, "2026-03-04": true, "2026-03-06": true, "2026-03-09": true, "2026-03-11": false} {
		if has(day, "MWF") != want {
			t.Errorf("MWF on %s = %v, want %v", day, !want, want)
		}
	}
	for day, want := range map[string]bool{"2026-03-01": true, "2026-03-02": false, "2026-03-03": true, "2026-03-05": false, "2026-03-07": true} {
		if has(day, "Alt") != want {
			t.Errorf("Alt on %s = %v, want %v", day, !want, want)
		}
	}
	for day, want := range map[string]bool{"2026-02-15": true, "2026-03-15": true, "2026-04-15": false, "2026-03-14": false} {
		if has(day, "Rent") != want {
			t.Errorf("Rent on %s = %v, want %v", day, !want, want)
		}
	}
}

func TestRecurrenceKeepsWallClockAcrossDST(t *testing.T) {
	// Berlin switches to summer time on 2026-03-29; a 09:00 series stays at 09:00 local.
	cal := Parse(wrap("UID:a\r\nSUMMARY:Daily\r\nDTSTART;TZID=Europe/Berlin:20260327T090000\r\nDTEND;TZID=Europe/Berlin:20260327T100000\r\nRRULE:FREQ=DAILY\r\n"), berlin)
	for _, day := range []string{"2026-03-27", "2026-03-30"} {
		if got := spans(cal.On(day, berlin)); !reflect.DeepEqual(got, [][3]any{{"Daily", 540, 600}}) {
			t.Errorf("%s: %v", day, got)
		}
	}
}

func TestRecurrenceOverride(t *testing.T) {
	cal := Parse(wrap(
		"UID:a\r\nSUMMARY:Sync\r\nDTSTART:20260302T090000\r\nDTEND:20260302T100000\r\nRRULE:FREQ=DAILY;COUNT=3\r\n",
		"UID:a\r\nSUMMARY:Sync (moved)\r\nRECURRENCE-ID:20260303T090000\r\nDTSTART:20260303T150000\r\nDTEND:20260303T160000\r\n",
	), berlin)
	if got := spans(cal.On("2026-03-03", berlin)); !reflect.DeepEqual(got, [][3]any{{"Sync (moved)", 900, 960}}) {
		t.Fatalf("got %v", got)
	}
}

func TestNormalize(t *testing.T) {
	for in, want := range map[string]string{"": "", "webcal://x.test/a.ics": "https://x.test/a.ics", " https://x.test/a.ics ": "https://x.test/a.ics"} {
		if got, err := Normalize(in); err != nil || got != want {
			t.Errorf("Normalize(%q) = %q, %v", in, got, err)
		}
	}
	for _, bad := range []string{"file:///etc/passwd", "ftp://x.test/a", "not a url", "javascript:alert(1)"} {
		if _, err := Normalize(bad); err == nil {
			t.Errorf("Normalize(%q) should fail", bad)
		}
	}
}

func TestFeedCachesAndSurvivesOutage(t *testing.T) {
	hits, fail := 0, false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		if fail {
			http.Error(w, "boom", http.StatusInternalServerError)
			return
		}
		w.Write([]byte(wrap("SUMMARY:Standup\r\nDTSTART:20260301T093000\r\nDTEND:20260301T100000\r\n")))
	}))
	defer srv.Close()

	now := time.Date(2026, 3, 1, 8, 0, 0, 0, berlin)
	f := NewFeed(srv.Client())
	f.now = func() time.Time { return now }
	if err := f.SetURL(srv.URL); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if es, err := f.Busy(ctx, "2026-03-01", berlin); err != nil || len(es) != 1 {
		t.Fatalf("first: %v %v", es, err)
	}
	f.Busy(ctx, "2026-03-01", berlin)
	if hits != 1 {
		t.Fatalf("second read should hit the cache, hits = %d", hits)
	}
	now = now.Add(10 * time.Minute)
	fail = true
	es, err := f.Busy(ctx, "2026-03-01", berlin)
	if err == nil || len(es) != 1 {
		t.Fatalf("outage should keep the old data and report an error: %v %v", es, err)
	}
	if f.Status() == nil {
		t.Fatal("status should carry the error")
	}
	if err := f.SetURL(""); err != nil || f.Configured() {
		t.Fatal("clearing the url disables the feed")
	}
}

func TestFeedRejectsNonCalendar(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("<html>login</html>")) }))
	defer srv.Close()
	f := NewFeed(srv.Client())
	_ = f.SetURL(srv.URL)
	if err := f.Refresh(context.Background()); err == nil {
		t.Fatal("expected an error for a non-ICS body")
	}
}
