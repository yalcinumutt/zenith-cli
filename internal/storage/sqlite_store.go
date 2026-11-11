package storage

import (
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schemaSQL string

// queryer is the subset of *sql.DB and *sql.Tx the store runs statements on, so
// the same methods work inside and outside a transaction.
type queryer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

type SQLiteStore struct {
	db *sql.DB
	q  queryer
	tx bool
}

// ErrNotFound is returned by single-row lookups that match nothing.
var ErrNotFound = errors.New("not found")

// timeLayout matches CURRENT_TIMESTAMP so injected times and database defaults
// compare and aggregate consistently.
const timeLayout = "2006-01-02 15:04:05"

func formatTime(t time.Time) string { return t.UTC().Format(timeLayout) }

// WithTx runs fn against a store bound to one transaction and commits when fn
// returns nil. Nested calls join the outer transaction.
func (s *SQLiteStore) WithTx(fn func(Store) error) error {
	if s.tx {
		return fn(s)
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	if err := fn(&SQLiteStore{db: s.db, q: tx, tx: true}); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func NewSQLiteStore() (*SQLiteStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("could not get home directory: %w", err)
	}

	dbPath := filepath.Join(home, ".zenith", "data.db")
	return NewSQLiteStoreAtPath(dbPath)
}

// openDB is the only place the database is opened. Pragmas ride on the DSN so
// every pooled connection gets them (ADR 0008).
func openDB(dbPath string) (*sql.DB, error) {
	q := url.Values{}
	q.Add("_pragma", "journal_mode(WAL)")
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "synchronous(NORMAL)")
	// The path must be URI-escaped: SQLite would otherwise read '?', '#' or '%'
	// in a home directory as DSN syntax and open the wrong file.
	p := filepath.ToSlash(dbPath)
	if filepath.VolumeName(dbPath) != "" {
		p = "/" + p // C:/x.db -> /C:/x.db
	}
	dsn := "file:" + (&url.URL{Path: p}).EscapedPath() + "?" + q.Encode()

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("could not open database: %w", err)
	}
	// One connection per process: SQLite serializes writers anyway, and it
	// keeps PRAGMA data_version meaningful - it then changes only when another
	// process writes, which is how the core detects external changes.
	db.SetMaxOpenConns(1)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not open database: %w", err)
	}
	return db, nil
}

func NewSQLiteStoreAtPath(dbPath string) (*SQLiteStore, error) {
	dbDir := filepath.Dir(dbPath)

	if _, err := os.Stat(dbDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dbDir, 0755); err != nil {
			return nil, fmt.Errorf("could not create database directory: %w", err)
		}
	}

	db, err := openDB(dbPath)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(schemaSQL); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not execute schema: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not run migrations: %w", err)
	}

	if err := ensureDefaultProject(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("could not ensure default project: %w", err)
	}

	return &SQLiteStore{db: db, q: db}, nil
}

func ensureDefaultProject(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM projects").Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		_, err := db.Exec("INSERT INTO projects (name, description) VALUES (?, ?)", "Anytime", "Default project for tasks")
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *SQLiteStore) AddTask(task *models.Task) error {
	query := `INSERT INTO tasks (project_id, title, description, status, priority, due_date, planned_date, recurring) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := s.q.Exec(query, task.ProjectID, task.Title, task.Description, task.Status, task.Priority, task.DueDate, task.PlannedDate, task.Recurring)
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

const taskSelect = `
		SELECT 
			t.id, t.project_id, t.title, t.description, t.status, t.priority, t.due_date, t.planned_date, t.recurring, t.created_at, t.updated_at,
			EXISTS(SELECT 1 FROM task_time_logs WHERE task_id = t.id AND end_time IS NULL) as is_running,
			COALESCE((SELECT SUM(strftime('%s', COALESCE(end_time, CURRENT_TIMESTAMP)) - strftime('%s', start_time)) FROM task_time_logs WHERE task_id = t.id), 0) as total_time
		FROM tasks t`

func (s *SQLiteStore) queryTasks(where string, args ...any) ([]models.Task, error) {
	rows, err := s.q.Query(taskSelect+where, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query tasks: %w", err)
	}

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		err := rows.Scan(&t.ID, &t.ProjectID, &t.Title, &t.Description, &t.Status, &t.Priority, &t.DueDate, &t.PlannedDate, &t.Recurring, &t.CreatedAt, &t.UpdatedAt, &t.IsRunning, &t.TotalTime)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("could not scan task: %w", err)
		}
		tasks = append(tasks, t)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	// Release the connection before the per-task tag lookups.
	rows.Close()

	for i := range tasks {
		tags, err := s.GetTagsForTask(tasks[i].ID)
		if err != nil {
			return nil, fmt.Errorf("could not get tags for task %d: %w", tasks[i].ID, err)
		}
		tasks[i].Tags = tags
	}
	return tasks, nil
}

func (s *SQLiteStore) GetTasks() ([]models.Task, error) {
	return s.queryTasks("")
}

// GetTask returns ErrNotFound when no task has the given id.
func (s *SQLiteStore) GetTask(id int64) (*models.Task, error) {
	tasks, err := s.queryTasks(" WHERE t.id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, ErrNotFound
	}
	return &tasks[0], nil
}

// UpdateTask persists every task column and stamps updated_at with
// task.UpdatedAt (or the database clock when it is zero).
func (s *SQLiteStore) UpdateTask(task *models.Task) error {
	updated := any(nil)
	if !task.UpdatedAt.IsZero() {
		updated = formatTime(task.UpdatedAt)
	}
	query := `UPDATE tasks SET project_id = ?, title = ?, description = ?, status = ?, priority = ?, due_date = ?, planned_date = ?, recurring = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?`
	_, err := s.q.Exec(query, task.ProjectID, task.Title, task.Description, task.Status, task.Priority, task.DueDate, task.PlannedDate, task.Recurring, updated, task.ID)
	if err != nil {
		return fmt.Errorf("could not update task: %w", err)
	}
	return nil
}

func (s *SQLiteStore) DeleteTask(id int64) error {
	_, err := s.q.Exec("DELETE FROM tasks WHERE id = ?", id)
	return err
}

func (s *SQLiteStore) AddHabit(habit *models.Habit) error {
	query := `INSERT INTO habits (name, description, frequency) VALUES (?, ?, ?)`
	res, err := s.q.Exec(query, habit.Name, habit.Description, habit.Frequency)
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
	rows, err := s.q.Query("SELECT id, name, description, frequency, created_at, updated_at FROM habits")
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
	_, err := s.q.Exec("INSERT INTO habit_logs (habit_id) VALUES (?)", habitID)
	return err
}

func (s *SQLiteStore) AddProject(project *models.Project) error {
	res, err := s.q.Exec("INSERT INTO projects (name, description) VALUES (?, ?)", project.Name, project.Description)
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
	rows, err := s.q.Query("SELECT id, name, description, created_at, updated_at FROM projects")
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
	term := "%" + query + "%"
	return s.queryTasks(" WHERE t.title LIKE ? OR t.description LIKE ?", term, term)
}

func (s *SQLiteStore) SearchHabits(query string) ([]models.Habit, error) {
	sqlQuery := `SELECT id, name, description, frequency, created_at, updated_at FROM habits WHERE name LIKE ? OR description LIKE ?`
	searchTerm := "%" + query + "%"
	rows, err := s.q.Query(sqlQuery, searchTerm, searchTerm)
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

// OpenTimeLog starts a time log for the task at the given instant.
func (s *SQLiteStore) OpenTimeLog(taskID int64, start time.Time) error {
	_, err := s.q.Exec(`INSERT INTO task_time_logs (task_id, start_time) VALUES (?, ?)`, taskID, formatTime(start))
	return err
}

// CloseOpenTimeLog ends the task's open time log, if any, and reports how many
// seconds it ran. closed is false when the task had no open log.
func (s *SQLiteStore) CloseOpenTimeLog(taskID int64, end time.Time) (secs int64, closed bool, err error) {
	var logID, startUnix int64
	err = s.q.QueryRow(`SELECT id, CAST(strftime('%s', start_time) AS INTEGER) FROM task_time_logs WHERE task_id = ? AND end_time IS NULL`, taskID).Scan(&logID, &startUnix)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if _, err = s.q.Exec(`UPDATE task_time_logs SET end_time = ? WHERE id = ?`, formatTime(end), logID); err != nil {
		return 0, false, err
	}
	return end.Unix() - startUnix, true, nil
}

func (s *SQLiteStore) AddTag(tag *models.Tag) error {
	res, err := s.q.Exec(`INSERT INTO tags (name, color) VALUES (?, ?) ON CONFLICT(name) DO UPDATE SET color = excluded.color`, tag.Name, tag.Color)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err == nil && id != 0 {
		tag.ID = id
	} else if id == 0 {
		// If it was an update or already exists, fetch the ID
		err = s.q.QueryRow(`SELECT id FROM tags WHERE name = ?`, tag.Name).Scan(&tag.ID)
	}
	return err
}

func (s *SQLiteStore) AttachTagToTask(taskID, tagID int64) error {
	_, err := s.q.Exec(`INSERT OR IGNORE INTO task_tags (task_id, tag_id) VALUES (?, ?)`, taskID, tagID)
	return err
}

func (s *SQLiteStore) GetTagsForTask(taskID int64) ([]models.Tag, error) {
	rows, err := s.q.Query(`
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

func (s *SQLiteStore) AddScheduledTimer(timer *models.ScheduledTimer) error {
	query := `INSERT INTO scheduled_timers (task_id, title, trigger_time, status) VALUES (?, ?, ?, ?)`
	res, err := s.q.Exec(query, timer.TaskID, timer.Title, timer.TriggerTime, "pending")
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	timer.ID = id
	return nil
}

func (s *SQLiteStore) GetPendingScheduledTimers() ([]models.ScheduledTimer, error) {
	query := `SELECT id, task_id, title, trigger_time, status, created_at FROM scheduled_timers WHERE status = 'pending'`
	rows, err := s.q.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var timers []models.ScheduledTimer
	for rows.Next() {
		var t models.ScheduledTimer
		err := rows.Scan(&t.ID, &t.TaskID, &t.Title, &t.TriggerTime, &t.Status, &t.CreatedAt)
		if err != nil {
			return nil, err
		}
		timers = append(timers, t)
	}
	return timers, nil
}

func (s *SQLiteStore) ClaimScheduledTimer(id int64) (bool, error) {
	res, err := s.q.Exec(`UPDATE scheduled_timers SET status = 'triggered' WHERE id = ? AND status = 'pending'`, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	return n == 1, err
}

func (s *SQLiteStore) AddTaskHistory(history *models.TaskHistory) error {
	stamp := any(nil)
	if !history.Timestamp.IsZero() {
		stamp = formatTime(history.Timestamp)
	}
	query := `INSERT INTO task_history (task_id, action, details, timestamp) VALUES (?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP))`
	res, err := s.q.Exec(query, history.TaskID, history.Action, history.Details, stamp)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	history.ID = id
	return nil
}

func (s *SQLiteStore) GetTaskHistory(taskID int64) ([]models.TaskHistory, error) {
	query := `SELECT id, task_id, action, details, timestamp FROM task_history WHERE task_id = ? ORDER BY id DESC`
	rows, err := s.q.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []models.TaskHistory
	for rows.Next() {
		var h models.TaskHistory
		err := rows.Scan(&h.ID, &h.TaskID, &h.Action, &h.Details, &h.Timestamp)
		if err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// DataVersion reports PRAGMA data_version. The pool is limited to one
// connection (see openDB), so this value moves only for writes made by other
// processes, never for this store's own.
func (s *SQLiteStore) DataVersion() (int64, error) {
	var v int64
	err := s.q.QueryRow("PRAGMA data_version").Scan(&v)
	return v, err
}
