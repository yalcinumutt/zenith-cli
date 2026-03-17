package storage

import (
	"github.com/yalcinumut/zenith-cli/internal/models"
)

type Store interface {
	// Tasks
	AddTask(task *models.Task) error
	GetTasks() ([]models.Task, error)
	UpdateTask(task *models.Task) error
	DeleteTask(id int64) error

	// Habits
	AddHabit(habit *models.Habit) error
	GetHabits() ([]models.Habit, error)
	LogHabit(habitID int64) error

	// Projects
	AddProject(project *models.Project) error
	GetProjects() ([]models.Project, error)

	// Search
	SearchTasks(query string) ([]models.Task, error)
	SearchHabits(query string) ([]models.Habit, error)
}
