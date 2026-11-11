package appapi

import "github.com/yalcinumutt/zenith-cli/internal/models"

// DTOs are the plain shapes the desktop front end sees. They carry no core types,
// no time.Time and no pointers to entities: dates are YYYY-MM-DD strings,
// instants are RFC 3339 strings. Every entity carries both its integer handle
// (the CLI's id, used to address it) and its sync uuid.

type Info struct {
	Version string `json:"version"`
	Beta    bool   `json:"beta"`
	Today   string `json:"today"` // the day "today" means, YYYY-MM-DD
}

type Counts struct {
	Inbox    int `json:"inbox"`
	Today    int `json:"today"`
	Upcoming int `json:"upcoming"`
}

type TagDTO struct {
	ID    int64  `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

type RepeatDTO struct {
	Kind     string `json:"kind"` // days | weeks | months
	Interval int    `json:"interval"`
	Weekdays []int  `json:"weekdays"` // 0=Sunday .. 6=Saturday
	MonthDay int    `json:"monthDay"`
	Text     string `json:"text"` // human description, e.g. "Every 2 weeks on Mon, Wed"
}

type TaskDTO struct {
	ID             int64      `json:"id"`
	UUID           string     `json:"uuid"`
	Title          string     `json:"title"`
	Notes          string     `json:"notes"`
	Status         string     `json:"status"`   // open | completed | cancelled
	WhenKind       string     `json:"whenKind"` // anytime | scheduled | someday
	WhenDate       string     `json:"whenDate"`
	Deadline       string     `json:"deadline"`
	Evening        bool       `json:"evening"`
	ProjectID      *int64     `json:"projectId"`
	ProjectName    string     `json:"projectName"`
	AreaID         *int64     `json:"areaId"`
	HeadingID      *int64     `json:"headingId"`
	ParentID       *int64     `json:"parentId"`
	Tags           []TagDTO   `json:"tags"`
	Repeat         *RepeatDTO `json:"repeat"`
	SubtasksOpen   int        `json:"subtasksOpen"`
	SubtasksTotal  int        `json:"subtasksTotal"`
	Running        bool       `json:"running"`
	SecondsTracked int64      `json:"secondsTracked"`
	Overdue        bool       `json:"overdue"` // open, with a Deadline before today
	UpdatedAt      string     `json:"updatedAt"`
}

type TodayDTO struct {
	Day     []TaskDTO `json:"day"`
	Evening []TaskDTO `json:"evening"`
}

type ProjectDTO struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	Name      string `json:"name"`
	AreaID    *int64 `json:"areaId"`
	Status    string `json:"status"`
	OpenTasks int    `json:"openTasks"`
}

type AreaDTO struct {
	ID       int64        `json:"id"`
	UUID     string       `json:"uuid"`
	Name     string       `json:"name"`
	Projects []ProjectDTO `json:"projects"`
}

type SidebarDTO struct {
	Areas      []AreaDTO    `json:"areas"`
	Standalone []ProjectDTO `json:"standalone"` // projects with no area
}

type HeadingDTO struct {
	ID        int64  `json:"id"`
	UUID      string `json:"uuid"`
	ProjectID int64  `json:"projectId"`
	Title     string `json:"title"`
}

// ProjectViewDTO is everything a project page needs in one call.
type ProjectViewDTO struct {
	Project  ProjectDTO   `json:"project"`
	Headings []HeadingDTO `json:"headings"`
	Tasks    []TaskDTO    `json:"tasks"`
}

// AreaViewDTO is an area page: its own tasks and its projects.
type AreaViewDTO struct {
	Area  AreaDTO   `json:"area"`
	Tasks []TaskDTO `json:"tasks"`
}

type ReminderDTO struct {
	ID     int64  `json:"id"`
	TaskID *int64 `json:"taskId"`
	Title  string `json:"title"`
	At     string `json:"at"` // RFC 3339
}

// NewTask is the input for creating a task. Zero values mean "not set".
type NewTask struct {
	Title     string   `json:"title"`
	Notes     string   `json:"notes"`
	WhenKind  string   `json:"whenKind"` // "", anytime, scheduled, someday
	WhenDate  string   `json:"whenDate"`
	Deadline  string   `json:"deadline"`
	Evening   bool     `json:"evening"`
	ProjectID *int64   `json:"projectId"`
	AreaID    *int64   `json:"areaId"`
	HeadingID *int64   `json:"headingId"`
	ParentID  *int64   `json:"parentId"`
	Tags      []string `json:"tags"`
}

// TaskPatch changes only the fields that are set. A field is cleared by its
// explicit Clear flag or an empty string where that is meaningful.
type TaskPatch struct {
	Title        *string   `json:"title"`
	Notes        *string   `json:"notes"`
	WhenKind     *string   `json:"whenKind"`
	WhenDate     *string   `json:"whenDate"`
	Deadline     *string   `json:"deadline"` // "" clears
	Evening      *bool     `json:"evening"`
	Tags         *[]string `json:"tags"`
	ProjectID    *int64    `json:"projectId"`
	ClearProject bool      `json:"clearProject"` // also clears area and heading
	AreaID       *int64    `json:"areaId"`
	HeadingID    *int64    `json:"headingId"`
	ClearHeading bool      `json:"clearHeading"`
}

// Place is a drag/keyboard move: neighbours in the list being shown, plus an
// optional heading or evening change applied in the same single write.
type Place struct {
	AfterID    *int64 `json:"afterId"`
	BeforeID   *int64 `json:"beforeId"`
	SetHeading bool   `json:"setHeading"`
	HeadingID  *int64 `json:"headingId"` // with SetHeading; nil removes the heading
	SetEvening bool   `json:"setEvening"`
	Evening    bool   `json:"evening"`
}

// Prefs are the app's own preferences, stored by the app (not in the web view).
type Prefs struct {
	Theme    string `json:"theme"`    // system | light | dark
	LastView string `json:"lastView"` // opaque to Go; the front end's view key
}

// ChangeEvent is what the shell forwards to the front end for every core event.
type ChangeEvent struct {
	Type   string   `json:"type"` // task | lists | reminder | external
	TaskID int64    `json:"taskId"`
	Kind   string   `json:"kind"`
	Keys   []string `json:"keys"`
}

var _ = models.StatusOpen
