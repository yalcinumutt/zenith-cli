package core

import "github.com/yalcinumutt/zenith-cli/internal/models"

// SearchResult holds the matches for one query.
type SearchResult struct {
	Tasks  []models.Task
	Habits []models.Habit
}

// Search finds tasks and habits whose title/name or description contain query.
func (c *Core) Search(query string) (*SearchResult, error) {
	tasks, err := c.store.SearchTasks(query)
	if err != nil {
		return nil, err
	}
	habits, err := c.store.SearchHabits(query)
	if err != nil {
		return nil, err
	}
	return &SearchResult{Tasks: tasks, Habits: habits}, nil
}

// Overview is everything the daily summary and the AI prompts read.
type Overview struct {
	Projects []models.Project
	Tasks    []models.Task
	Habits   []models.Habit
}

// Overview loads projects, tasks and habits in one call.
func (c *Core) Overview() (*Overview, error) {
	projects, err := c.Projects.List()
	if err != nil {
		return nil, err
	}
	tasks, err := c.Tasks.List()
	if err != nil {
		return nil, err
	}
	habits, err := c.Habits.List()
	if err != nil {
		return nil, err
	}
	return &Overview{Projects: projects, Tasks: tasks, Habits: habits}, nil
}
