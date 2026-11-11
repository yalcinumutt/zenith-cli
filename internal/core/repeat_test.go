package core_test

import (
	"testing"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

func day(s string) time.Time {
	d, err := time.Parse(models.DateLayout, s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestNextOccurrenceRules(t *testing.T) {
	days := func(n int) models.RepeatRule { return models.RepeatRule{Kind: models.RepeatDays, Interval: n} }
	weeks := func(n int, wd ...int) models.RepeatRule {
		return models.RepeatRule{Kind: models.RepeatWeeks, Interval: n, Weekdays: wd}
	}
	months := func(n, d int) models.RepeatRule {
		return models.RepeatRule{Kind: models.RepeatMonths, Interval: n, MonthDay: d}
	}
	cases := []struct {
		name string
		rule models.RepeatRule
		from string
		want string
	}{
		{"daily", days(1), "2026-03-01", "2026-03-02"},
		{"every 3 days across month end", days(3), "2026-02-27", "2026-03-02"},
		{"daily over a leap day", days(1), "2028-02-28", "2028-02-29"},
		{"every week, same weekday", weeks(1), "2026-03-02", "2026-03-09"},
		{"mon/wed from monday", weeks(1, 1, 3), "2026-03-02", "2026-03-04"},       // Mon -> Wed
		{"mon/wed from wednesday", weeks(1, 1, 3), "2026-03-04", "2026-03-09"},    // Wed -> next Mon
		{"mon/wed from friday", weeks(1, 1, 3), "2026-03-06", "2026-03-09"},       // Fri -> next Mon
		{"sunday counts as week end", weeks(1, 0, 1), "2026-03-01", "2026-03-02"}, // Sun -> Mon
		{"mon from sunday", weeks(1, 1), "2026-03-01", "2026-03-02"},
		{"every 2 weeks on fri from fri", weeks(2, 5), "2026-03-06", "2026-03-20"},
		{"every 2 weeks mon/thu from mon", weeks(2, 1, 4), "2026-03-02", "2026-03-05"},
		{"every 2 weeks mon/thu from thu", weeks(2, 1, 4), "2026-03-05", "2026-03-16"},
		{"monthly on the 15th", months(1, 15), "2026-01-15", "2026-02-15"},
		{"monthly on the 31st clamps to Feb", months(1, 31), "2026-01-31", "2026-02-28"},
		{"monthly on the 31st recovers", months(1, 31), "2026-02-28", "2026-03-31"},
		{"monthly on the 31st into a 30-day month", months(1, 31), "2026-03-31", "2026-04-30"},
		{"monthly 30th over a leap February", months(1, 30), "2028-01-30", "2028-02-29"},
		{"every 3 months across the year", months(3, 15), "2026-11-15", "2027-02-15"},
		{"monthly without a day uses the date's day", months(1, 0), "2026-05-20", "2026-06-20"},
		{"every 12 months on Feb 29 anchor", months(12, 29), "2028-02-29", "2029-02-28"},
	}
	for _, c := range cases {
		got := core.NextOccurrence(c.rule, day(c.from))
		if got.Format(models.DateLayout) != c.want {
			t.Errorf("%s: next after %s = %s, want %s", c.name, c.from, got.Format(models.DateLayout), c.want)
		}
		if !got.After(day(c.from)) {
			t.Errorf("%s: %s is not after %s", c.name, got, c.from)
		}
	}
}

func TestRuleDescriptions(t *testing.T) {
	for want, rule := range map[string]models.RepeatRule{
		"Every day":                {Kind: models.RepeatDays, Interval: 1},
		"Every 3 days":             {Kind: models.RepeatDays, Interval: 3},
		"Every week on Mon, Wed":   {Kind: models.RepeatWeeks, Interval: 1, Weekdays: []int{1, 3}},
		"Every 2 weeks on Fri":     {Kind: models.RepeatWeeks, Interval: 2, Weekdays: []int{5}},
		"Every month on day 15":    {Kind: models.RepeatMonths, Interval: 1, MonthDay: 15},
		"Every 3 months on day 31": {Kind: models.RepeatMonths, Interval: 3, MonthDay: 31},
	} {
		if got := rule.String(); got != want {
			t.Errorf("String() = %q, want %q", got, want)
		}
	}
	if days, err := core.ParseWeekdays("Mon, wed,FRI"); err != nil || len(days) != 3 || days[0] != 1 || days[2] != 5 {
		t.Errorf("ParseWeekdays = %v, %v", days, err)
	}
}

func repeating(t *testing.T, c *core.Core, title, date string, rule models.RepeatRule) *models.Task {
	t.Helper()
	return add(t, c, core.AddTaskInput{Title: title, When: sched(date), Repeat: &rule})
}

func TestCompletingAnOccurrenceCreatesExactlyTheNext(t *testing.T) {
	c, _ := newCore(t)
	first := repeating(t, c, "water plants", "2026-03-01", models.RepeatRule{Kind: models.RepeatDays, Interval: 2})
	_ = c.Tasks.Tag(first.ID, "home", "")

	if err := c.Tasks.Complete(first.ID); err != nil {
		t.Fatal(err)
	}
	all, _ := c.Tasks.List()
	if len(all) != 2 {
		t.Fatalf("tasks = %v, want the completed one plus the next", titles(all))
	}
	next := all[1]
	if next.Status != models.StatusOpen || next.When != sched("2026-03-03") || next.OccurrenceDate != "2026-03-03" ||
		next.RepeaterID == nil || *next.RepeaterID != *first.RepeaterID || len(next.Tags) != 1 || next.Title != "water plants" {
		t.Errorf("next = %+v", next)
	}
	// The completed occurrence stays in the Logbook; the next is in Upcoming.
	if got := list(t, c, core.ListLogbook); len(got) != 1 {
		t.Errorf("logbook = %v", got)
	}
	if got := list(t, c, core.ListUpcoming); len(got) != 1 {
		t.Errorf("upcoming = %v", got)
	}
	if h, _ := c.Tasks.History(next.ID); len(h) != 1 || h[0].Action != "created" {
		t.Errorf("history = %+v", h)
	}
}

func TestCompletingTwiceNeverDuplicatesTheNextOccurrence(t *testing.T) {
	c, _ := newCore(t)
	first := repeating(t, c, "stretch", "2026-03-01", models.RepeatRule{Kind: models.RepeatDays, Interval: 1})

	_ = c.Tasks.Complete(first.ID)
	_ = c.Tasks.Complete(first.ID) // already completed: nothing happens
	if all, _ := c.Tasks.List(); len(all) != 2 {
		t.Fatalf("double complete produced %d tasks", len(all))
	}
	// Reopening and completing again must not mint another next occurrence either.
	_ = c.Tasks.Reopen(first.ID)
	_ = c.Tasks.Complete(first.ID)
	if all, _ := c.Tasks.List(); len(all) != 2 {
		t.Fatalf("reopen+complete produced %d tasks", len(all))
	}
	// Completing the next occurrence continues the series.
	all, _ := c.Tasks.List()
	_ = c.Tasks.Complete(all[1].ID)
	if all, _ := c.Tasks.List(); len(all) != 3 || all[2].OccurrenceDate != "2026-03-03" {
		t.Errorf("series = %+v", titles(all))
	}
}

func TestWeekdaySetSeriesAndMonthEnds(t *testing.T) {
	c, _ := newCore(t)
	task := repeating(t, c, "gym", "2026-03-02", models.RepeatRule{Kind: models.RepeatWeeks, Interval: 1, Weekdays: []int{1, 3, 5}}) // Mon
	var dates []string
	cur := task
	for i := 0; i < 5; i++ {
		_ = c.Tasks.Complete(cur.ID)
		all, _ := c.Tasks.List()
		cur = &all[len(all)-1]
		dates = append(dates, cur.OccurrenceDate)
	}
	want := []string{"2026-03-04", "2026-03-06", "2026-03-09", "2026-03-11", "2026-03-13"}
	for i := range want {
		if dates[i] != want[i] {
			t.Fatalf("dates = %v, want %v", dates, want)
		}
	}

	m := repeating(t, c, "rent", "2026-01-31", models.RepeatRule{Kind: models.RepeatMonths, Interval: 1, MonthDay: 31})
	var got []string
	cur = m
	for i := 0; i < 4; i++ {
		_ = c.Tasks.Complete(cur.ID)
		all, _ := c.Tasks.List()
		cur = &all[len(all)-1]
		got = append(got, cur.OccurrenceDate)
	}
	if want := []string{"2026-02-28", "2026-03-31", "2026-04-30", "2026-05-31"}; !equal(got, want) {
		t.Errorf("month-end series = %v, want %v", got, want)
	}
}

func equal(a, b []string) bool {
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

func TestDeadlineOffsetAndTemplateFollowTheLastOccurrence(t *testing.T) {
	c, _ := newCore(t)
	proj, _ := c.Projects.Add("Home", nil)
	first := add(t, c, core.AddTaskInput{
		Title: "pay bill", When: sched("2026-03-01"), Deadline: "2026-03-04", ProjectID: &proj.ID,
		Repeat: &models.RepeatRule{Kind: models.RepeatMonths, Interval: 1},
	})
	_ = c.Tasks.Complete(first.ID)
	all, _ := c.Tasks.List()
	next := all[1]
	if next.Deadline != "2026-04-04" || next.When != sched("2026-04-01") || next.ProjectID == nil || *next.ProjectID != proj.ID {
		t.Errorf("next = %+v", next)
	}
	// An edit to the occurrence carries forward to the one after.
	_, _ = c.Tasks.Get(next.ID)
	_ = c.Tasks.SetDeadline(next.ID, "2026-04-10")
	_ = c.Tasks.Complete(next.ID)
	all, _ = c.Tasks.List()
	if after := all[2]; after.Deadline != "2026-05-10" {
		t.Errorf("deadline offset not carried: %+v", after)
	}
}

func TestSetChangeAndRemoveARule(t *testing.T) {
	c, _ := newCore(t)
	task := add(t, c, core.AddTaskInput{Title: "plain"}) // anytime, never repeated

	if r, _ := c.Repeaters.ForTask(task.ID); r != nil {
		t.Fatal("should not repeat yet")
	}
	if err := c.Repeaters.Set(task.ID, models.RepeatRule{Kind: models.RepeatWeeks, Interval: 2, Weekdays: []int{5, 1}}); err != nil {
		t.Fatal(err)
	}
	got, _ := c.Tasks.Get(task.ID)
	if got.When.Kind != models.WhenScheduled || got.OccurrenceDate != "2026-03-01" {
		t.Errorf("a repeating task needs a day: %+v", got)
	}
	r, _ := c.Repeaters.ForTask(task.ID)
	if r.Rule.String() != "Every 2 weeks on Mon, Fri" {
		t.Errorf("rule = %q", r.Rule)
	}
	// Change it.
	if err := c.Repeaters.Set(task.ID, models.RepeatRule{Kind: models.RepeatDays, Interval: 5}); err != nil {
		t.Fatal(err)
	}
	if r, _ := c.Repeaters.ForTask(task.ID); r.Rule.String() != "Every 5 days" {
		t.Errorf("rule = %q", r.Rule)
	}
	// Remove it: completing now produces nothing more.
	if err := c.Repeaters.Remove(task.ID); err != nil {
		t.Fatal(err)
	}
	_ = c.Tasks.Complete(task.ID)
	if all, _ := c.Tasks.List(); len(all) != 1 {
		t.Errorf("removed series still generated: %v", titles(all))
	}
	// Validation.
	bad := []models.RepeatRule{
		{Kind: "yearly", Interval: 1},
		{Kind: models.RepeatDays, Interval: -1},
		{Kind: models.RepeatWeeks, Interval: 1, Weekdays: []int{7}},
		{Kind: models.RepeatMonths, Interval: 1, MonthDay: 32},
	}
	for _, rule := range bad {
		if err := c.Repeaters.Set(task.ID, rule); err == nil {
			t.Errorf("rule %+v accepted", rule)
		}
	}
	if err := c.Repeaters.Set(99, models.RepeatRule{Kind: models.RepeatDays}); err == nil {
		t.Error("missing task accepted")
	}
}

func TestLegacyRecurringTasksMigrateToRepeaters(t *testing.T) {
	path := legacyDB(t, legacyV012)
	c := openAt(t, path)

	// Task 5 was weekly, anchored on its deadline 2026-03-05 (a Thursday).
	r, err := c.Repeaters.ForTask(5)
	if err != nil || r == nil {
		t.Fatalf("task 5 repeater: %v, %v", r, err)
	}
	if r.Rule.Kind != models.RepeatWeeks || r.Rule.Interval != 1 || len(r.Rule.Weekdays) != 1 || r.Rule.Weekdays[0] != 4 || r.Title != "Ship landing page" {
		t.Errorf("rule = %+v", r)
	}
	if got, _ := c.Tasks.Get(5); got.OccurrenceDate != "2026-03-05" {
		t.Errorf("occurrence = %q", got.OccurrenceDate)
	}
	for _, id := range []int64{1, 2, 6, 7} {
		if r, _ := c.Repeaters.ForTask(id); r != nil {
			t.Errorf("task %d should not repeat", id)
		}
	}
	// recurring is gone, and a second open changes nothing.
	db := rawDB(t, path)
	var n int
	_ = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('tasks') WHERE name = 'recurring'").Scan(&n)
	db.Close()
	if n != 0 {
		t.Error("recurring column still exists")
	}
	c2 := openAt(t, path)
	var count int
	rdb := rawDB(t, path)
	_ = rdb.QueryRow("SELECT COUNT(*) FROM repeaters").Scan(&count)
	rdb.Close()
	if count != 1 {
		t.Errorf("repeaters after reopen = %d", count)
	}

	// Completing the migrated occurrence generates the following Thursday.
	if err := c2.Tasks.Complete(5); err != nil {
		t.Fatal(err)
	}
	all, _ := c2.Tasks.List()
	next := all[len(all)-1]
	if next.OccurrenceDate != "2026-03-12" || next.Title != "Ship landing page" {
		t.Errorf("next = %+v", next)
	}
	if got := list(t, c2, core.ListLogbook); len(got) < 2 {
		t.Errorf("logbook = %v", got)
	}
}
