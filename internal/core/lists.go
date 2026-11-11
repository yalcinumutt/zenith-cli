package core

import (
	"sort"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// ListKind names one of the six lists. They are derived views, never stored
// (ADR 0004): every front-end gets identical results from the same functions.
type ListKind string

const (
	ListInbox    ListKind = "inbox"
	ListToday    ListKind = "today"
	ListUpcoming ListKind = "upcoming"
	ListAnytime  ListKind = "anytime"
	ListSomeday  ListKind = "someday"
	ListLogbook  ListKind = "logbook"
)

// ListKinds is the sidebar order.
var ListKinds = []ListKind{ListInbox, ListToday, ListUpcoming, ListAnytime, ListSomeday, ListLogbook}

// ListService computes the derived lists from When, Deadline and Status.
type ListService struct {
	store storage.Store
	clock Clock
}

// TodayList is Today split into its two sections.
type TodayList struct {
	Day     []models.Task
	Evening []models.Task
}

// Counts are the badge numbers shown beside Inbox, Today and Upcoming.
type Counts struct {
	Inbox    int `json:"inbox"`
	Today    int `json:"today"`
	Upcoming int `json:"upcoming"`
}

// ToDay is the calendar day "today" means right now, in the clock's zone.
func (s *ListService) ToDay() string { return s.clock.Now().Format(models.DateLayout) }

func isOpenTop(t models.Task) bool { return t.Status == models.StatusOpen && t.ParentID == nil }

func hasContainer(t models.Task) bool {
	return t.ProjectID != nil || t.AreaID != nil || t.HeadingID != nil
}

// inToday: scheduled for today or earlier (an unfinished Today task stays in
// Today when its day passes), or with a Deadline of today or earlier.
func inToday(t models.Task, today string) bool {
	if !isOpenTop(t) {
		return false
	}
	if t.When.Kind == models.WhenScheduled && t.When.Date <= today {
		return true
	}
	return t.Deadline != "" && t.Deadline <= today
}

func member(kind ListKind, t models.Task, today string) bool {
	switch kind {
	case ListToday:
		return inToday(t, today)
	case ListInbox:
		return isOpenTop(t) && !hasContainer(t) && t.When.Kind == models.WhenAnytime && !inToday(t, today)
	case ListUpcoming:
		return isOpenTop(t) && t.When.Kind == models.WhenScheduled && t.When.Date > today && !inToday(t, today)
	case ListAnytime:
		return isOpenTop(t) && t.When.Kind == models.WhenAnytime && hasContainer(t) && !inToday(t, today)
	case ListSomeday:
		return isOpenTop(t) && t.When.Kind == models.WhenSomeday && !inToday(t, today)
	case ListLogbook:
		return t.ParentID == nil && (t.Status == models.StatusCompleted || t.Status == models.StatusCancelled)
	}
	return false
}

// Get returns one list. Subtasks never appear: they show nested under their
// parent, and in their own Project's list (see Project).
func (s *ListService) Get(kind ListKind) ([]models.Task, error) {
	all, err := s.store.GetTasks()
	if err != nil {
		return nil, err
	}
	today := s.ToDay()
	var out []models.Task
	for _, t := range all {
		if member(kind, t, today) {
			out = append(out, t)
		}
	}
	switch kind {
	case ListUpcoming:
		sort.SliceStable(out, func(i, j int) bool { return out[i].When.Date < out[j].When.Date })
	case ListLogbook:
		sort.SliceStable(out, func(i, j int) bool {
			if !out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
				return out[i].UpdatedAt.After(out[j].UpdatedAt)
			}
			return out[i].ID > out[j].ID
		})
	}
	return out, nil
}

// Today returns the Today list split into the day and "This Evening".
func (s *ListService) Today() (TodayList, error) {
	tasks, err := s.Get(ListToday)
	if err != nil {
		return TodayList{}, err
	}
	var l TodayList
	for _, t := range tasks {
		if t.Evening && t.When.Kind == models.WhenScheduled {
			l.Evening = append(l.Evening, t)
		} else {
			l.Day = append(l.Day, t)
		}
	}
	return l, nil
}

// Counts returns the sidebar badge numbers.
func (s *ListService) Counts() (Counts, error) {
	all, err := s.store.GetTasks()
	if err != nil {
		return Counts{}, err
	}
	today := s.ToDay()
	var c Counts
	for _, t := range all {
		switch {
		case member(ListToday, t, today):
			c.Today++
		case member(ListInbox, t, today):
			c.Inbox++
		case member(ListUpcoming, t, today):
			c.Upcoming++
		}
	}
	return c, nil
}

// Project returns the open tasks filed in a project, including Subtasks that
// carry that project as their own. The caller nests them by ParentID.
func (s *ListService) Project(projectID int64) ([]models.Task, error) {
	tasks, err := s.store.GetTasksInProject(projectID)
	if err != nil {
		return nil, err
	}
	return openOnly(tasks), nil
}

// Area returns the open tasks filed directly in an area.
func (s *ListService) Area(areaID int64) ([]models.Task, error) {
	tasks, err := s.store.GetTasksInArea(areaID)
	if err != nil {
		return nil, err
	}
	return openOnly(tasks), nil
}

func openOnly(tasks []models.Task) []models.Task {
	var out []models.Task
	for _, t := range tasks {
		if t.Status == models.StatusOpen {
			out = append(out, t)
		}
	}
	return out
}
