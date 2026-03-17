package storage

import (
	"database/sql"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
	"github.com/yalcinumut/zenith-cli/internal/models"
)

//go:embed schema.sql
var schemaSQL string

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore() (*SQLiteStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get home directory: %w", err)
	}

	dbPath := filepath.Join(home, ".zenith", "data.db")
	return NewSQLiteStoreAtPath(dbPath)
}

func NewSQLiteStoreAtPath(dbPath string) (*SQLiteStore, error) {
	dbDir := filepath.Dir(dbPath)

	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("could not create database directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not execute schema: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not run migrations: %w", err)
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) AddTask(task *models.Task) error {
	query := `INSERT INTO tasks (project_id, title, description, status, priority, due_date, planned_date, recurring) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := s.db.Exec(query, task.ProjectID, task.Title, task.Description, task.Status, task.Priority, task.DueDate, task.PlannedDate, task.Recurring)
	if err != nil {
		return fmt.Errorf("could not insert task: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("could not get last insert id: %w", err)
	}

	task.ID = id
	return nil
}

func (s *SQLiteStore) GetTasks() ([]models.Task, error) {
	query := `
		SELECT 
			t.id, t.project_id, t.title, t.description, t.status, t.priority, t.due_date, t.planned_date, t.recurring, t.created_at, t.updated_at,
			EXISTS(SELECT 1 FROM task_time_logs WHERE task_id = t.id AND end_time IS NULL) as is_running,
			COALESCE((SELECT SUM(strftime('%s', COALESCE(end_time, CURRENT_TIMESTAMP)) - strftime('%s', start_time)) FROM task_time_logs WHERE task_id = t.id), 0) as total_time
		FROM tasks t`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("could not query tasks: %w", err)
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.PlannedDate, &t.Recurring, &t.CreatedAt, &t.UpdatedAt, &t.IsRunning, &t.TotalTime)
		if err != nil {
			return nil, fmt.Errorf("could not scan task: %w", err)
		}

		// Fetch tags for task
		tags, err := s.GetTagsForTask(t.ID)
		if err != nil {
			return nil, fmt.Errorf("could not get tags for task %d: %w", t.ID, err)
		}
		t.Tags = tags

		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *SQLiteStore) UpdateTask(task *models.Task) error {
	query := `UPDATE tasks SET project_id = ?, title = ?, description = ?, status = ?, priority = ?, due_date = ?, planned_date = ?, recurring = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`
	_, err := s.db.Exec(query, task.ProjectID, task.Title, task.Description, task.Status, task.Priority, task.DueDate, task.PlannedDate, task.Recurring, task.ID)
	if err != nil {
		return fmt.Errorf("could not update task: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteTask(id int64) error {
	_, err := s.db.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) AddHabit(habit *models.Habit) error {
	query := `INSERT INTO habits (name, description, frequency) VALUES (?, ?, ?)`
	res, err := s.db.Exec(query, habit.Name, habit.Description, habit.Frequency)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	habit.ID = id
	return nil
}

func (s *SQLiteStore) GetHabits() ([]models.Habit, error) {
	rows, err := s.db.Query("SELECT id, name, description, frequency, created_at, updated_at FROM habits")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []models.Habit
	for rows.Next() {
		var h models.Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.Description, &h.Frequency, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, nil
}

func (s *SQLiteStore) LogHabit(habitID int64) error {
	_, err := s.db.Exec("INSERT INTO habit_logs (habit_id) VALUES (?)", habitID)
	return err
}

func (s *SQLiteStore) AddProject(project *models.Project) error {
	res, err := s.db.Exec("INSERT INTO projects (name, description) VALUES (?, ?)", project.Name, project.Description)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	project.ID = id
	return nil
}

func (s *SQLiteStore) GetProjects() ([]models.Project, error) {
	rows, err := s.db.Query("SELECT id, name, description, created_at, updated_at FROM projects")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		if err := rows.Scan(&p.ID, &p.Name, &p.Description, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *SQLiteStore) SearchTasks(query string) ([]models.Task, error) {
	sqlQuery := `
		SELECT 
			t.id, t.project_id, t.title, t.description, t.status, t.priority, t.due_date, t.planned_date, t.recurring, t.created_at, t.updated_at,
			EXISTS(SELECT 1 FROM task_time_logs WHERE task_id = t.id AND end_time IS NULL) as is_running,
			COALESCE((SELECT SUM(strftime('%s', COALESCE(end_time, CURRENT_TIMESTAMP)) - strftime('%s', start_time)) FROM task_time_logs WHERE task_id = t.id), 0) as total_time
		FROM tasks t 
		WHERE t.title LIKE ? OR t.description LIKE ?`
	searchTerm := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, searchTerm, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		if err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.PlannedDate, &t.Recurring, &t.CreatedAt, &t.UpdatedAt, &t.IsRunning, &t.TotalTime); err != nil {
			return nil, err
		}

		// Fetch tags for task
		tags, err := s.GetTagsForTask(t.ID)
		if err != nil {
			return nil, fmt.Errorf("could not get tags for search task %d: %w", t.ID, err)
		}
		t.Tags = tags

		tasks = append(tasks, t)
	}
	return tasks, nil
}

func (s *SQLiteStore) SearchHabits(query string) ([]models.Habit, error) {
	sqlQuery := `SELECT id, name, description, frequency, created_at, updated_at FROM habits WHERE name LIKE ? OR description LIKE ?`
	searchTerm := "%" + query + "%"
	rows, err := s.db.Query(sqlQuery, searchTerm, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []models.Habit
	for rows.Next() {
		var h models.Habit
		if err := rows.Scan(&h.ID, &h.Name, &h.Description, &h.Frequency, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, nil
}

func (s *SQLiteStore) StartTaskTimer(taskID int64) error {
	// First stop any running timers for this task
	if err := s.StopTaskTimer(taskID); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO task_time_logs (task_id, start_time) VALUES (?, CURRENT_TIMESTAMP)`, taskID)
	return err
}

func (s *SQLiteStore) StopTaskTimer(taskID int64) error {
	_, err := s.db.Exec(`UPDATE task_time_logs SET end_time = CURRENT_TIMESTAMP WHERE task_id = ? AND end_time IS NULL`, taskID)
	return err
}

func (s *SQLiteStore) AddTag(tag *models.Tag) error {
	res, err := s.db.Exec(`INSERT INTO tags (name, color) VALUES (?, ?) ON CONFLICT(name) DO UPDATE SET color = excluded.color`, tag.Name, tag.Color)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil && id != 0 {
		tag.ID = id
	} else if id == 0 {
		// If it was an update or already exists, fetch the ID
		err = s.db.QueryRow(`SELECT id FROM tags WHERE name = ?`, tag.Name).Scan(&tag.ID)
	}
	return err
}

func (s *SQLiteStore) AttachTagToTask(taskID, tagID int64) error {
	_, err := s.db.Exec(`INSERT OR IGNORE INTO task_tags (task_id, tag_id) VALUES (?, ?)`, taskID, tagID)
	return err
}

func (s *SQLiteStore) GetTagsForTask(taskID int64) ([]models.Tag, error) {
	rows, err := s.db.Query(`
		SELECT t.id, t.name, t.color, t.created_at 
		FROM tags t
		JOIN task_tags tt ON t.id = tt.tag_id
		WHERE tt.task_id = ?`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
