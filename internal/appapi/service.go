// Package appapi is the adapter between the shared core and the desktop front
// end (ADR 0002). Its Service is what the Wails shell binds: plain methods over
// plain DTOs, no framework imports, so it is tested without a window. Methods
// return errors as ordinary Go errors; the shell turns them into rejected
// promises carrying the core's message (for example the refusal to complete a
// task that still has open subtasks).
package appapi

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
)

// Service is the bound API object.
type Service struct {
	core    *core.Core
	clock   core.Clock
	prefs   *prefsStore
	version string
}

// New builds the service. prefsPath may be empty (preferences then stay at defaults).
func New(c *core.Core, clock core.Clock, prefsPath, version string) *Service {
	if clock == nil {
		clock = core.SystemClock{}
	}
	return &Service{core: c, clock: clock, prefs: &prefsStore{path: prefsPath}, version: version}
}

func (s *Service) today() string { return s.clock.Now().Format(models.DateLayout) }

// ---- meta and preferences

func (s *Service) Info() Info {
	return Info{Version: s.version, Beta: true, Today: s.today()}
}

func (s *Service) GetPrefs() Prefs { return s.prefs.load() }

func (s *Service) SetPrefs(p Prefs) (Prefs, error) {
	if p.Theme != "light" && p.Theme != "dark" {
		p.Theme = "system"
	}
	if strings.TrimSpace(p.LastView) == "" {
		p.LastView = defaultPrefs.LastView
	}
	return p, s.prefs.save(p)
}

// ---- conversions

// index is a snapshot used to decorate task rows without a query per row.
type index struct {
	today    string
	projects map[int64]string
	kids     map[int64][2]int // parent id -> [open, total]
	repeats  map[int64]*models.Repeater
}

func (s *Service) index() (*index, error) {
	all, err := s.core.Tasks.List()
	if err != nil {
		return nil, err
	}
	projects, err := s.core.Projects.List()
	if err != nil {
		return nil, err
	}
	ix := &index{today: s.today(), projects: map[int64]string{}, kids: map[int64][2]int{}, repeats: map[int64]*models.Repeater{}}
	for _, p := range projects {
		ix.projects[p.ID] = p.Name
	}
	for _, t := range all {
		if t.ParentID != nil {
			c := ix.kids[*t.ParentID]
			if t.Status == models.StatusOpen {
				c[0]++
			}
			c[1]++
			ix.kids[*t.ParentID] = c
		}
	}
	return ix, nil
}

func (s *Service) repeat(ix *index, t models.Task) *RepeatDTO {
	if t.RepeaterID == nil {
		return nil
	}
	r, cached := ix.repeats[*t.RepeaterID]
	if !cached {
		rep, err := s.core.Repeaters.ForTask(t.ID)
		if err == nil {
			r = rep
		}
		ix.repeats[*t.RepeaterID] = r
	}
	if r == nil {
		return nil
	}
	days := r.Rule.Weekdays
	if days == nil {
		days = []int{}
	}
	return &RepeatDTO{Kind: string(r.Rule.Kind), Interval: r.Rule.Interval, Weekdays: days, MonthDay: r.Rule.MonthDay, Text: r.Rule.String()}
}

func (s *Service) toDTO(ix *index, t models.Task) TaskDTO {
	tags := make([]TagDTO, 0, len(t.Tags))
	for _, g := range t.Tags {
		tags = append(tags, TagDTO{ID: g.ID, Name: g.Name, Color: g.Color})
	}
	d := TaskDTO{
		ID: t.ID, UUID: t.UUID, Title: t.Title, Notes: t.Description, Status: t.Status,
		WhenKind: string(t.When.Kind), WhenDate: t.When.Date, Deadline: t.Deadline, Evening: t.Evening,
		ProjectID: t.ProjectID, AreaID: t.AreaID, HeadingID: t.HeadingID, ParentID: t.ParentID,
		Tags: tags, Repeat: s.repeat(ix, t), Running: t.IsRunning, SecondsTracked: t.TotalTime,
		Overdue:   t.Status == models.StatusOpen && t.Deadline != "" && t.Deadline < ix.today,
		UpdatedAt: t.UpdatedAt.Format(time.RFC3339),
	}
	if t.ProjectID != nil {
		d.ProjectName = ix.projects[*t.ProjectID]
	}
	if c, ok := ix.kids[t.ID]; ok {
		d.SubtasksOpen, d.SubtasksTotal = c[0], c[1]
	}
	return d
}

func (s *Service) toDTOs(tasks []models.Task) ([]TaskDTO, error) {
	ix, err := s.index()
	if err != nil {
		return nil, err
	}
	out := make([]TaskDTO, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, s.toDTO(ix, t))
	}
	return out, nil
}

// ---- lists

func (s *Service) Counts() (Counts, error) {
	c, err := s.core.Lists.Counts()
	return Counts{Inbox: c.Inbox, Today: c.Today, Upcoming: c.Upcoming}, err
}

// List returns one of inbox, upcoming, anytime, someday or logbook (use Today for today).
func (s *Service) List(kind string) ([]TaskDTO, error) {
	k := core.ListKind(strings.ToLower(kind))
	valid := false
	for _, known := range core.ListKinds {
		valid = valid || known == k
	}
	if !valid {
		return nil, fmt.Errorf("unknown list %q", kind)
	}
	tasks, err := s.core.Lists.Get(k)
	if err != nil {
		return nil, err
	}
	return s.toDTOs(tasks)
}

func (s *Service) Today() (TodayDTO, error) {
	t, err := s.core.Lists.Today()
	if err != nil {
		return TodayDTO{}, err
	}
	day, err := s.toDTOs(t.Day)
	if err != nil {
		return TodayDTO{}, err
	}
	evening, err := s.toDTOs(t.Evening)
	return TodayDTO{Day: day, Evening: evening}, err
}

func (s *Service) projectDTO(p models.Project, open int) ProjectDTO {
	return ProjectDTO{ID: p.ID, UUID: p.UUID, Name: p.Name, AreaID: p.AreaID, Status: p.Status, OpenTasks: open}
}

func (s *Service) Sidebar() (SidebarDTO, error) {
	areas, err := s.core.Areas.List()
	if err != nil {
		return SidebarDTO{}, err
	}
	projects, err := s.core.Projects.List()
	if err != nil {
		return SidebarDTO{}, err
	}
	all, err := s.core.Tasks.List()
	if err != nil {
		return SidebarDTO{}, err
	}
	open := map[int64]int{}
	for _, t := range all {
		if t.Status == models.StatusOpen && t.ProjectID != nil && t.ParentID == nil {
			open[*t.ProjectID]++
		}
	}
	sb := SidebarDTO{Areas: []AreaDTO{}, Standalone: []ProjectDTO{}}
	byArea := map[int64]int{}
	for _, a := range areas {
		byArea[a.ID] = len(sb.Areas)
		sb.Areas = append(sb.Areas, AreaDTO{ID: a.ID, UUID: a.UUID, Name: a.Name, Projects: []ProjectDTO{}})
	}
	for _, p := range projects {
		if p.Status != models.StatusOpen {
			continue
		}
		dto := s.projectDTO(p, open[p.ID])
		if p.AreaID != nil {
			if i, ok := byArea[*p.AreaID]; ok {
				sb.Areas[i].Projects = append(sb.Areas[i].Projects, dto)
				continue
			}
		}
		sb.Standalone = append(sb.Standalone, dto)
	}
	return sb, nil
}

func (s *Service) ProjectView(id int64) (ProjectViewDTO, error) {
	p, err := s.core.Projects.Get(id)
	if err != nil {
		return ProjectViewDTO{}, err
	}
	hs, err := s.core.Headings.List(id)
	if err != nil {
		return ProjectViewDTO{}, err
	}
	tasks, err := s.core.Lists.Project(id)
	if err != nil {
		return ProjectViewDTO{}, err
	}
	dtos, err := s.toDTOs(tasks)
	if err != nil {
		return ProjectViewDTO{}, err
	}
	out := ProjectViewDTO{Project: s.projectDTO(*p, len(tasks)), Headings: []HeadingDTO{}, Tasks: dtos}
	for _, h := range hs {
		out.Headings = append(out.Headings, HeadingDTO{ID: h.ID, UUID: h.UUID, ProjectID: h.ProjectID, Title: h.Title})
	}
	return out, nil
}

func (s *Service) AreaView(id int64) (AreaViewDTO, error) {
	a, err := s.core.Areas.Get(id)
	if err != nil {
		return AreaViewDTO{}, err
	}
	tasks, err := s.core.Lists.Area(id)
	if err != nil {
		return AreaViewDTO{}, err
	}
	dtos, err := s.toDTOs(tasks)
	if err != nil {
		return AreaViewDTO{}, err
	}
	sb, err := s.Sidebar()
	if err != nil {
		return AreaViewDTO{}, err
	}
	view := AreaViewDTO{Area: AreaDTO{ID: a.ID, UUID: a.UUID, Name: a.Name, Projects: []ProjectDTO{}}, Tasks: dtos}
	for _, ar := range sb.Areas {
		if ar.ID == id {
			view.Area.Projects = ar.Projects
		}
	}
	return view, nil
}

// ---- tasks

func (s *Service) GetTask(id int64) (TaskDTO, error) {
	t, err := s.core.Tasks.Get(id)
	if err != nil {
		return TaskDTO{}, err
	}
	ix, err := s.index()
	if err != nil {
		return TaskDTO{}, err
	}
	return s.toDTO(ix, *t), nil
}

func (s *Service) Subtasks(id int64) ([]TaskDTO, error) {
	kids, err := s.core.Tasks.Subtasks(id)
	if err != nil {
		return nil, err
	}
	return s.toDTOs(kids)
}

func (s *Service) when(kind, date string) (models.When, error) {
	switch models.WhenKind(kind) {
	case "", models.WhenAnytime:
		return models.When{Kind: models.WhenAnytime}, nil
	case models.WhenSomeday:
		return models.When{Kind: models.WhenSomeday}, nil
	case models.WhenScheduled:
		return models.When{Kind: models.WhenScheduled, Date: date}, nil
	}
	return models.When{}, fmt.Errorf("unknown when %q", kind)
}

func (s *Service) AddTask(in NewTask) (TaskDTO, error) {
	w, err := s.when(in.WhenKind, in.WhenDate)
	if err != nil {
		return TaskDTO{}, err
	}
	title := strings.TrimSpace(in.Title)
	if title == "" {
		return TaskDTO{}, errors.New("a task needs a title")
	}
	t, err := s.core.Tasks.Add(core.AddTaskInput{
		Title: title, When: w, Deadline: in.Deadline, Evening: in.Evening,
		ProjectID: in.ProjectID, AreaID: in.AreaID, HeadingID: in.HeadingID, ParentID: in.ParentID,
		Priority: models.PriorityMedium,
	})
	if err != nil {
		return TaskDTO{}, err
	}
	if in.Notes != "" {
		if err := s.core.Tasks.Edit(t.ID, core.TaskEdit{Description: &in.Notes}); err != nil {
			return TaskDTO{}, err
		}
	}
	if len(in.Tags) > 0 {
		if err := s.core.Tasks.SetTags(t.ID, in.Tags); err != nil {
			return TaskDTO{}, err
		}
	}
	return s.GetTask(t.ID)
}

func (s *Service) UpdateTask(id int64, p TaskPatch) (TaskDTO, error) {
	if p.Title != nil || p.Notes != nil {
		if err := s.core.Tasks.Edit(id, core.TaskEdit{Title: p.Title, Description: p.Notes}); err != nil {
			return TaskDTO{}, err
		}
	}
	if p.WhenKind != nil {
		date := ""
		if p.WhenDate != nil {
			date = *p.WhenDate
		}
		w, err := s.when(*p.WhenKind, date)
		if err != nil {
			return TaskDTO{}, err
		}
		if err := s.core.Tasks.SetWhen(id, w); err != nil {
			return TaskDTO{}, err
		}
	}
	if p.Evening != nil {
		if err := s.core.Tasks.SetEvening(id, *p.Evening); err != nil {
			return TaskDTO{}, err
		}
	}
	if p.Deadline != nil {
		if err := s.core.Tasks.SetDeadline(id, *p.Deadline); err != nil {
			return TaskDTO{}, err
		}
	}
	if p.Tags != nil {
		if err := s.core.Tasks.SetTags(id, *p.Tags); err != nil {
			return TaskDTO{}, err
		}
	}
	switch {
	case p.ClearProject:
		if err := s.core.Tasks.Unfile(id); err != nil {
			return TaskDTO{}, err
		}
	case p.ProjectID != nil:
		if err := s.core.Tasks.MoveToProject(id, *p.ProjectID); err != nil {
			return TaskDTO{}, err
		}
	case p.AreaID != nil:
		if err := s.core.Tasks.FileInArea(id, *p.AreaID); err != nil {
			return TaskDTO{}, err
		}
	}
	switch {
	case p.ClearHeading:
		if err := s.core.Tasks.SetHeading(id, nil); err != nil {
			return TaskDTO{}, err
		}
	case p.HeadingID != nil:
		if err := s.core.Tasks.SetHeading(id, p.HeadingID); err != nil {
			return TaskDTO{}, err
		}
	}
	return s.GetTask(id)
}

func (s *Service) Complete(id int64) error   { return s.core.Tasks.Complete(id) }
func (s *Service) Reopen(id int64) error     { return s.core.Tasks.Reopen(id) }
func (s *Service) Cancel(id int64) error     { return s.core.Tasks.Cancel(id) }
func (s *Service) DeleteTask(id int64) error { return s.core.Tasks.Delete(id) }

func (s *Service) PlaceTask(id int64, p Place) error {
	pl := core.Placement{AfterID: p.AfterID, BeforeID: p.BeforeID}
	if p.SetHeading {
		pl.Heading = &core.HeadingChange{HeadingID: p.HeadingID}
	}
	if p.SetEvening {
		ev := p.Evening
		pl.Evening = &ev
	}
	return s.core.Tasks.Place(id, pl)
}

func (s *Service) Tags() ([]TagDTO, error) {
	tags, err := s.core.Tags.List()
	out := make([]TagDTO, 0, len(tags))
	for _, g := range tags {
		out = append(out, TagDTO{ID: g.ID, Name: g.Name, Color: g.Color})
	}
	return out, err
}

// ---- repeating

func (s *Service) SetRepeat(taskID int64, r RepeatDTO) (TaskDTO, error) {
	rule := models.RepeatRule{Kind: models.RepeatKind(r.Kind), Interval: r.Interval, Weekdays: r.Weekdays, MonthDay: r.MonthDay}
	if err := s.core.Repeaters.Set(taskID, rule); err != nil {
		return TaskDTO{}, err
	}
	return s.GetTask(taskID)
}

func (s *Service) RemoveRepeat(taskID int64) (TaskDTO, error) {
	if err := s.core.Repeaters.Remove(taskID); err != nil {
		return TaskDTO{}, err
	}
	return s.GetTask(taskID)
}

// ---- areas, projects, headings

func (s *Service) AddArea(name string) (AreaDTO, error) {
	a, err := s.core.Areas.Add(name)
	if err != nil {
		return AreaDTO{}, err
	}
	return AreaDTO{ID: a.ID, UUID: a.UUID, Name: a.Name, Projects: []ProjectDTO{}}, nil
}
func (s *Service) RenameArea(id int64, name string) error { return s.core.Areas.Rename(id, name) }
func (s *Service) DeleteArea(id int64) error              { return s.core.Areas.Delete(id) }

func (s *Service) AddProject(name string, areaID *int64) (ProjectDTO, error) {
	p, err := s.core.Projects.Add(name, areaID)
	if err != nil {
		return ProjectDTO{}, err
	}
	return s.projectDTO(*p, 0), nil
}
func (s *Service) RenameProject(id int64, name string) error { return s.core.Projects.Rename(id, name) }
func (s *Service) MoveProject(id int64, areaID *int64) error { return s.core.Projects.Move(id, areaID) }

// PlaceProject files a project under an area (or makes it standalone) and puts
// it between two neighbours, in one drag. SetArea says whether AreaID applies.
func (s *Service) PlaceProject(id int64, setArea bool, areaID *int64, afterID, beforeID *int64) error {
	if setArea {
		if err := s.core.Projects.Move(id, areaID); err != nil {
			return err
		}
	}
	if afterID == nil && beforeID == nil {
		return nil
	}
	return s.core.Projects.Reorder(id, afterID, beforeID)
}

func (s *Service) ReorderArea(id int64, afterID, beforeID *int64) error {
	return s.core.Areas.Reorder(id, afterID, beforeID)
}

func (s *Service) ReorderHeading(id int64, afterID, beforeID *int64) error {
	return s.core.Headings.Reorder(id, afterID, beforeID)
}

func (s *Service) CompleteProject(id int64) error { return s.core.Projects.Complete(id) }
func (s *Service) CancelProject(id int64) error   { return s.core.Projects.Cancel(id) }
func (s *Service) ReopenProject(id int64) error   { return s.core.Projects.Reopen(id) }
func (s *Service) DeleteProject(id int64) error   { return s.core.Projects.Delete(id) }

func (s *Service) AddHeading(projectID int64, title string) (HeadingDTO, error) {
	h, err := s.core.Headings.Add(projectID, title)
	if err != nil {
		return HeadingDTO{}, err
	}
	return HeadingDTO{ID: h.ID, UUID: h.UUID, ProjectID: h.ProjectID, Title: h.Title}, nil
}
func (s *Service) RenameHeading(id int64, title string) error {
	return s.core.Headings.Rename(id, title)
}
func (s *Service) DeleteHeading(id int64) error { return s.core.Headings.Delete(id) }

// ---- reminders

// AddReminder schedules a reminder for an instant (RFC 3339), optionally on a task.
func (s *Service) AddReminder(taskID *int64, title, at string) (ReminderDTO, error) {
	when, err := time.Parse(time.RFC3339, at)
	if err != nil {
		return ReminderDTO{}, fmt.Errorf("invalid time %q: %w", at, err)
	}
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Reminder"
	}
	d := when.Sub(s.clock.Now())
	if d < 0 {
		d = 0
	}
	r, err := s.core.Reminders.Add(title, d, taskID)
	if err != nil {
		return ReminderDTO{}, err
	}
	return ReminderDTO{ID: r.ID, TaskID: r.TaskID, Title: r.Title, At: r.TriggerTime.Format(time.RFC3339)}, nil
}

func (s *Service) PendingReminders() ([]ReminderDTO, error) {
	rs, err := s.core.Reminders.Pending()
	out := make([]ReminderDTO, 0, len(rs))
	for _, r := range rs {
		out = append(out, ReminderDTO{ID: r.ID, TaskID: r.TaskID, Title: r.Title, At: r.TriggerTime.Format(time.RFC3339)})
	}
	return out, err
}

// ---- events

// ToChangeEvent converts a core event for the front end; ok is false for events it does not forward.
func ToChangeEvent(e core.Event) (ChangeEvent, bool) {
	switch v := e.(type) {
	case core.TaskChanged:
		return ChangeEvent{Type: "task", TaskID: v.TaskID, Kind: string(v.Kind), Keys: []string{}}, true
	case core.ListsInvalidated:
		return ChangeEvent{Type: "lists", Keys: v.Keys}, true
	case core.ReminderDue:
		return ChangeEvent{Type: "reminder", TaskID: v.ReminderID, Keys: []string{}}, true
	case core.ExternalChange:
		return ChangeEvent{Type: "external", Keys: []string{}}, true
	}
	return ChangeEvent{}, false
}

// habitHistoryDays is how far back the habit views can look: the heatmap's 26
// weeks plus room to page back through months on the calendar.
const habitHistoryDays = 400

func (s *Service) Habits() ([]HabitDTO, error) {
	hs, err := s.core.Habits.List()
	if err != nil {
		return nil, err
	}
	days, err := s.core.Habits.Days(habitHistoryDays)
	if err != nil {
		return nil, err
	}
	out := make([]HabitDTO, 0, len(hs))
	for _, h := range hs {
		d := days[h.ID]
		if d == nil {
			d = []string{}
		}
		out = append(out, HabitDTO{ID: h.ID, UUID: h.UUID, Name: h.Name, Days: d})
	}
	return out, nil
}

func (s *Service) AddHabit(name string) (HabitDTO, error) {
	h, err := s.core.Habits.Add(name)
	if err != nil {
		return HabitDTO{}, err
	}
	return HabitDTO{ID: h.ID, UUID: h.UUID, Name: h.Name, Days: []string{}}, nil
}

func (s *Service) RenameHabit(id int64, name string) error { return s.core.Habits.Rename(id, name) }
func (s *Service) DeleteHabit(id int64) error              { return s.core.Habits.Delete(id) }

// SetHabitDay ticks or un-ticks a habit on one day (YYYY-MM-DD, not in the future).
func (s *Service) SetHabitDay(id int64, day string, done bool) error {
	return s.core.Habits.SetDay(id, day, done)
}
