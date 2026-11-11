package storage

import (
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
)

// Store is plain persistence: it records what it is told and applies no
// business rules. Rules (history, status transitions, timer exclusivity,
// not-found errors) live in internal/core.
type Store interface {
	// WithTx runs fn against a store bound to one transaction.
	WithTx(fn func(Store) error) error

	// DataVersion is SQLite's change counter for this connection: it moves only
	// when a different connection or process commits a write.
	DataVersion() (int64, error)

	// Tasks
	AddTask(task *models.Task) error
	GetTasks() ([]models.Task, error)
	GetTask(id int64) (*models.Task, error) // ErrNotFound when absent
	UpdateTask(task *models.Task) error
	DeleteTask(id int64) error
	OpenTimeLog(taskID int64, start time.Time) error
	CloseOpenTimeLog(taskID int64, end time.Time) (secs int64, closed bool, err error)

	// Tags
	AddTag(tag *models.Tag) error
	AttachTagToTask(taskID, tagID int64) error
	GetTagsForTask(taskID int64) ([]models.Tag, error)

	// Habits
	AddHabit(habit *models.Habit) error
	GetHabits() ([]models.Habit, error)
	GetHabit(id int64) (*models.Habit, error) // ErrNotFound when absent
	LogHabit(habitID int64) error

	// Projects
	AddProject(project *models.Project) error
	GetProjects() ([]models.Project, error)

	// Search
	SearchTasks(query string) ([]models.Task, error)
	SearchHabits(query string) ([]models.Habit, error)

	// Scheduled Timers
	AddScheduledTimer(timer *models.ScheduledTimer) error
	GetPendingScheduledTimers() ([]models.ScheduledTimer, error)
	// ClaimScheduledTimer flips a pending timer to triggered in one guarded
	// UPDATE and reports whether this caller won; only one caller ever does.
	ClaimScheduledTimer(id int64) (bool, error)

	// Task History
	AddTaskHistory(history *models.TaskHistory) error
	GetTaskHistory(taskID int64) ([]models.TaskHistory, error)
}
