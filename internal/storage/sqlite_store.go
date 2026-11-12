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
	db   *sql.DB
	q    queryer
	tx   bool
	sync *syncCtx
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
	if err := fn(&SQLiteStore{db: s.db, q: tx, tx: true, sync: s.sync}); err != nil {
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

	return &SQLiteStore{db: db, q: db, sync: &syncCtx{now: time.Now}}, nil
}

func (s *SQLiteStore) addTask(task *models.Task) error {
	task.UUID = newUUID()
	query := `INSERT INTO tasks (uuid, project_id, area_id, heading_id, parent_task_id, title, description, status, priority, when_kind, when_date, deadline, evening, repeater_id, occurrence_date, sort_key) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	res, err := s.q.Exec(query, task.UUID, task.ProjectID, task.AreaID, task.HeadingID, task.ParentID, task.Title, task.Description, task.Status, task.Priority, whenKindOrDefault(task.When), nullIfEmpty(task.When.Date), nullIfEmpty(task.Deadline), task.Evening, task.RepeaterID, nullIfEmpty(task.OccurrenceDate), nullIfEmpty(task.SortKey))
	if err != nil {
		return fmt.Errorf("could not insert task: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return fmt.Errorf("could not get last insert id: %w", err)
	}

	task.ID = id
	return s.stamp("task", task.UUID, taskCols...)
}

// taskOrder is the one place task order is defined: fractional index, then uuid.
const taskOrder = " ORDER BY t.sort_key, t.uuid"

// taskSelect reads a live task. A container (project, area, heading or parent)
// that has been tombstoned reads as absent, so a live task under a deleted
// container is container-less until the core repairs it (ADR 0010).
const taskSelect = `
		SELECT 
			t.id, t.uuid, 
			CASE WHEN EXISTS(SELECT 1 FROM projects c WHERE c.id = t.project_id AND c.deleted_at IS NULL) THEN t.project_id END,
			CASE WHEN EXISTS(SELECT 1 FROM areas c WHERE c.id = t.area_id AND c.deleted_at IS NULL) THEN t.area_id END,
			CASE WHEN EXISTS(SELECT 1 FROM headings c JOIN projects hp ON hp.id = c.project_id WHERE c.id = t.heading_id AND c.deleted_at IS NULL AND hp.deleted_at IS NULL) THEN t.heading_id END,
			CASE WHEN EXISTS(SELECT 1 FROM tasks c WHERE c.id = t.parent_task_id AND c.deleted_at IS NULL) THEN t.parent_task_id END,
			t.title, t.description, t.status, t.priority, t.when_kind, t.when_date, t.deadline, t.evening, t.repeater_id, t.occurrence_date, t.sort_key, t.created_at, t.updated_at,
			EXISTS(SELECT 1 FROM task_time_logs WHERE task_id = t.id AND end_time IS NULL) as is_running,
			COALESCE((SELECT SUM(strftime('%s', COALESCE(end_time, CURRENT_TIMESTAMP)) - strftime('%s', start_time)) FROM task_time_logs WHERE task_id = t.id), 0) as total_time
		FROM tasks t`

// queryTasks runs the shared task select. cond is an extra condition (or ""),
// order an ORDER BY clause (or ""); tombstoned tasks are never returned.
func (s *SQLiteStore) queryTasks(cond, order string, args ...any) ([]models.Task, error) {
	where := " WHERE t.deleted_at IS NULL"
	if cond != "" {
		where += " AND (" + cond + ")"
	}
	rows, err := s.q.Query(taskSelect+where+order, args...)
	if err != nil {
		return nil, fmt.Errorf("could not query tasks: %w", err)
	}

	var tasks []models.Task
	for rows.Next() {
		var t models.Task
		var whenKind, whenDate, deadline sql.NullString
		var evening sql.NullBool
		var sortKey, occurrence sql.NullString
		err := rows.Scan(&t.ID, &t.UUID, &t.ProjectID, &t.AreaID, &t.HeadingID, &t.ParentID, &t.Title, &t.Description, &t.Status, &t.Priority, &whenKind, &whenDate, &deadline, &evening, &t.RepeaterID, &occurrence, &sortKey, &t.CreatedAt, &t.UpdatedAt, &t.IsRunning, &t.TotalTime)
		if err != nil {
			rows.Close()
			return nil, fmt.Errorf("could not scan task: %w", err)
		}
		t.When = models.When{Kind: models.WhenKind(whenKind.String), Date: whenDate.String}
		if t.When.Kind == "" {
			t.When.Kind = models.WhenAnytime
		}
		t.Deadline, t.Evening, t.SortKey = deadline.String, evening.Bool, sortKey.String
		t.OccurrenceDate = occurrence.String
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
	return s.queryTasks("", taskOrder)
}

// GetSubtasks returns the direct children of a task, oldest first.
func (s *SQLiteStore) GetSubtasks(parentID int64) ([]models.Task, error) {
	return s.queryTasks("t.parent_task_id = ?", taskOrder, parentID)
}

// GetTask returns ErrNotFound when no task has the given id.
func (s *SQLiteStore) GetTask(id int64) (*models.Task, error) {
	tasks, err := s.queryTasks("t.id = ?", "", id)
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
func (s *SQLiteStore) updateTask(task *models.Task) error {
	if err := s.stampDiff("tasks", "task", task.ID, taskCols, taskVals(task)); err != nil {
		return err
	}
	updated := any(nil)
	if !task.UpdatedAt.IsZero() {
		updated = formatTime(task.UpdatedAt)
	}
	query := `UPDATE tasks SET project_id = ?, area_id = ?, heading_id = ?, parent_task_id = ?, title = ?, description = ?, status = ?, priority = ?, when_kind = ?, when_date = ?, deadline = ?, evening = ?, repeater_id = ?, occurrence_date = ?, sort_key = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?`
	_, err := s.q.Exec(query, task.ProjectID, task.AreaID, task.HeadingID, task.ParentID, task.Title, task.Description, task.Status, task.Priority, whenKindOrDefault(task.When), nullIfEmpty(task.When.Date), nullIfEmpty(task.Deadline), task.Evening, task.RepeaterID, nullIfEmpty(task.OccurrenceDate), nullIfEmpty(task.SortKey), updated, task.ID)
	if err != nil {
		return fmt.Errorf("could not update task: %w", err)
	}
	return nil
}

// DeleteTask tombstones the task and the tag memberships that belong to it, and
// drops its device-local rows (reminders, a running time log). Finished time
// logs and history are grow-only and stay. Subtasks are the core's job.
func (s *SQLiteStore) deleteTask(id int64) error {
	var uuid string
	err := s.q.QueryRow("SELECT uuid FROM tasks WHERE id = ? AND deleted_at IS NULL", id).Scan(&uuid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	rows, err := s.q.Query(`SELECT tg.id, tg.uuid FROM task_tags tt JOIN tags tg ON tg.id = tt.tag_id WHERE tt.task_id = ? AND tt.deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	type member struct {
		id   int64
		uuid string
	}
	var members []member
	for rows.Next() {
		var m member
		if err := rows.Scan(&m.id, &m.uuid); err != nil {
			rows.Close()
			return err
		}
		members = append(members, m)
	}
	rows.Close()
	for _, m := range members {
		if err := s.detachTagFromTask(id, m.id); err != nil {
			return err
		}
	}
	for _, q := range []string{
		"DELETE FROM scheduled_timers WHERE task_id = ?",
		"DELETE FROM task_time_logs WHERE task_id = ? AND end_time IS NULL",
	} {
		if _, err := s.q.Exec(q, id); err != nil {
			return err
		}
	}
	return s.softDelete("tasks", "task", id)
}

func (s *SQLiteStore) addHabit(habit *models.Habit) error {
	habit.UUID = newUUID()
	query := `INSERT INTO habits (uuid, name, description, frequency) VALUES (?, ?, ?, ?)`
	res, err := s.q.Exec(query, habit.UUID, habit.Name, habit.Description, habit.Frequency)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	habit.ID = id
	return s.stamp("habit", habit.UUID, habitCols...)
}

func (s *SQLiteStore) GetHabits() ([]models.Habit, error) {
	rows, err := s.q.Query("SELECT id, uuid, name, description, frequency, created_at, updated_at FROM habits WHERE deleted_at IS NULL ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []models.Habit
	for rows.Next() {
		var h models.Habit
		if err := rows.Scan(&h.ID, &h.UUID, &h.Name, &h.Description, &h.Frequency, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, nil
}

func (s *SQLiteStore) GetHabit(id int64) (*models.Habit, error) {
	var h models.Habit
	err := s.q.QueryRow("SELECT id, uuid, name, description, frequency, created_at, updated_at FROM habits WHERE id = ? AND deleted_at IS NULL", id).
		Scan(&h.ID, &h.UUID, &h.Name, &h.Description, &h.Frequency, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (s *SQLiteStore) logHabit(habitID int64) error {
	u := newUUID()
	if _, err := s.q.Exec("INSERT INTO habit_logs (uuid, habit_id) VALUES (?, ?)", u, habitID); err != nil {
		return err
	}
	return s.stamp("habit_log", u, "_created")
}

func (s *SQLiteStore) updateHabit(h *models.Habit) error {
	if err := s.stampDiff("habits", "habit", h.ID, habitCols, map[string]string{"name": h.Name, "description": h.Description, "frequency": h.Frequency}); err != nil {
		return err
	}
	_, err := s.q.Exec("UPDATE habits SET name = ?, description = ?, frequency = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?", h.Name, h.Description, h.Frequency, stamp(h.UpdatedAt), h.ID)
	return err
}

func (s *SQLiteStore) logHabitAt(habitID int64, at time.Time) error {
	u := newUUID()
	if _, err := s.q.Exec("INSERT INTO habit_logs (uuid, habit_id, completed_at) VALUES (?, ?, ?)", u, habitID, formatTime(at)); err != nil {
		return err
	}
	return s.stamp("habit_log", u, "_created")
}

func (s *SQLiteStore) GetHabitLogs(from, to time.Time) ([]models.HabitLog, error) {
	rows, err := s.q.Query(`SELECT l.id, l.uuid, l.habit_id, l.completed_at FROM habit_logs l
		JOIN habits h ON h.id = l.habit_id AND h.deleted_at IS NULL
		WHERE l.deleted_at IS NULL AND l.completed_at >= ? AND l.completed_at < ?
		ORDER BY l.completed_at`, formatTime(from), formatTime(to))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var logs []models.HabitLog
	for rows.Next() {
		var l models.HabitLog
		if err := rows.Scan(&l.ID, &l.UUID, &l.HabitID, &l.CompletedAt); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *SQLiteStore) addProject(project *models.Project) error {
	project.UUID = newUUID()
	res, err := s.q.Exec("INSERT INTO projects (uuid, area_id, name, description, status, sort_key) VALUES (?, ?, ?, ?, ?, ?)", project.UUID, project.AreaID, project.Name, project.Description, project.Status, nullIfEmpty(project.SortKey))
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	project.ID = id
	return s.stamp("project", project.UUID, projectCols...)
}

func (s *SQLiteStore) GetProjects() ([]models.Project, error) {
	rows, err := s.q.Query("SELECT id, uuid, area_id, name, description, status, sort_key, created_at, updated_at FROM projects WHERE deleted_at IS NULL ORDER BY sort_key, uuid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []models.Project
	for rows.Next() {
		var p models.Project
		var sortKey sql.NullString
		if err := rows.Scan(&p.ID, &p.UUID, &p.AreaID, &p.Name, &p.Description, &p.Status, &sortKey, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, err
		}
		p.SortKey = sortKey.String
		projects = append(projects, p)
	}
	return projects, nil
}

func (s *SQLiteStore) SearchTasks(query string) ([]models.Task, error) {
	term := "%" + query + "%"
	return s.queryTasks("t.title LIKE ? OR t.description LIKE ?", taskOrder, term, term)
}

func (s *SQLiteStore) SearchHabits(query string) ([]models.Habit, error) {
	sqlQuery := `SELECT id, uuid, name, description, frequency, created_at, updated_at FROM habits WHERE deleted_at IS NULL AND (name LIKE ? OR description LIKE ?)`
	searchTerm := "%" + query + "%"
	rows, err := s.q.Query(sqlQuery, searchTerm, searchTerm)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var habits []models.Habit
	for rows.Next() {
		var h models.Habit
		if err := rows.Scan(&h.ID, &h.UUID, &h.Name, &h.Description, &h.Frequency, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		habits = append(habits, h)
	}
	return habits, nil
}

// OpenTimeLog starts a time log for the task at the given instant.
func (s *SQLiteStore) OpenTimeLog(taskID int64, start time.Time) error {
	_, err := s.q.Exec(`INSERT INTO task_time_logs (uuid, task_id, start_time) VALUES (?, ?, ?)`, newUUID(), taskID, formatTime(start))
	return err
}

// CloseOpenTimeLog ends the task's open time log, if any, and reports how many
// seconds it ran. closed is false when the task had no open log.
func (s *SQLiteStore) CloseOpenTimeLog(taskID int64, end time.Time) (secs int64, closed bool, err error) {
	var logID, startUnix int64
	var logUUID string
	err = s.q.QueryRow(`SELECT id, uuid, CAST(strftime('%s', start_time) AS INTEGER) FROM task_time_logs WHERE task_id = ? AND end_time IS NULL`, taskID).Scan(&logID, &logUUID, &startUnix)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if _, err = s.q.Exec(`UPDATE task_time_logs SET end_time = ? WHERE id = ?`, formatTime(end), logID); err != nil {
		return 0, false, err
	}
	// A time log becomes synced when it is stopped; while running it is device-local.
	if err = s.stamp("time_log", logUUID, "_created"); err != nil {
		return 0, false, err
	}
	return end.Unix() - startUnix, true, nil
}

// AddTag inserts a tag. Name uniqueness is the core's rule, not the database's.
func (s *SQLiteStore) addTag(tag *models.Tag) error {
	tag.UUID = newUUID()
	res, err := s.q.Exec(`INSERT INTO tags (uuid, name, color) VALUES (?, ?, ?)`, tag.UUID, tag.Name, tag.Color)
	if err != nil {
		return err
	}
	if tag.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	return s.stamp("tag", tag.UUID, tagCols...)
}

func (s *SQLiteStore) GetTags() ([]models.Tag, error) {
	rows, err := s.q.Query(`SELECT id, uuid, name, color, created_at FROM tags WHERE deleted_at IS NULL ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		var color sql.NullString
		if err := rows.Scan(&t.ID, &t.UUID, &t.Name, &color, &t.CreatedAt); err != nil {
			return nil, err
		}
		t.Color = color.String
		tags = append(tags, t)
	}
	return tags, rows.Err()
}

func (s *SQLiteStore) updateTag(tag *models.Tag) error {
	if err := s.stampDiff("tags", "tag", tag.ID, tagCols, tagVals(tag)); err != nil {
		return err
	}
	_, err := s.q.Exec(`UPDATE tags SET name = ?, color = ? WHERE id = ?`, tag.Name, tag.Color, tag.ID)
	return err
}

// membershipUUID identifies a task-tag membership for stamping.
func (s *SQLiteStore) membershipUUID(taskID, tagID int64) (string, error) {
	var tu, gu string
	if err := s.q.QueryRow("SELECT uuid FROM tasks WHERE id = ?", taskID).Scan(&tu); err != nil {
		return "", err
	}
	if err := s.q.QueryRow("SELECT uuid FROM tags WHERE id = ?", tagID).Scan(&gu); err != nil {
		return "", err
	}
	return tu + ":" + gu, nil
}

// DetachTagFromTask tombstones the membership (a stamped removal).
func (s *SQLiteStore) detachTagFromTask(taskID, tagID int64) error {
	res, err := s.q.Exec(`UPDATE task_tags SET deleted_at = ? WHERE task_id = ? AND tag_id = ? AND deleted_at IS NULL`, formatTime(s.sync.now()), taskID, tagID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return nil
	}
	mu, err := s.membershipUUID(taskID, tagID)
	if err != nil {
		return err
	}
	return s.stamp("task_tag", mu, "deleted_at")
}

// AttachTagToTask adds the membership, reviving a tombstoned one. Attaching a
// tag the task already has changes and stamps nothing.
func (s *SQLiteStore) attachTagToTask(taskID, tagID int64) error {
	var deleted sql.NullString
	err := s.q.QueryRow(`SELECT deleted_at FROM task_tags WHERE task_id = ? AND tag_id = ?`, taskID, tagID).Scan(&deleted)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		if _, err := s.q.Exec(`INSERT INTO task_tags (task_id, tag_id) VALUES (?, ?)`, taskID, tagID); err != nil {
			return err
		}
	case err != nil:
		return err
	case deleted.Valid:
		if _, err := s.q.Exec(`UPDATE task_tags SET deleted_at = NULL WHERE task_id = ? AND tag_id = ?`, taskID, tagID); err != nil {
			return err
		}
	default:
		return nil // already attached
	}
	mu, err := s.membershipUUID(taskID, tagID)
	if err != nil {
		return err
	}
	return s.stamp("task_tag", mu, "deleted_at")
}

func (s *SQLiteStore) GetTagsForTask(taskID int64) ([]models.Tag, error) {
	rows, err := s.q.Query(`
		SELECT t.id, t.uuid, t.name, t.color, t.created_at 
		FROM tags t
		JOIN task_tags tt ON t.id = tt.tag_id
		WHERE tt.task_id = ? AND tt.deleted_at IS NULL AND t.deleted_at IS NULL
		ORDER BY t.id`, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []models.Tag
	for rows.Next() {
		var t models.Tag
		if err := rows.Scan(&t.ID, &t.UUID, &t.Name, &t.Color, &t.CreatedAt); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func (s *SQLiteStore) AddScheduledTimer(timer *models.ScheduledTimer) error {
	timer.UUID = newUUID()
	query := `INSERT INTO scheduled_timers (uuid, task_id, title, trigger_time, status) VALUES (?, ?, ?, ?, ?)`
	res, err := s.q.Exec(query, timer.UUID, timer.TaskID, timer.Title, timer.TriggerTime, "pending")
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
	query := `SELECT id, uuid, task_id, title, trigger_time, status, created_at FROM scheduled_timers WHERE status = 'pending'`
	rows, err := s.q.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var timers []models.ScheduledTimer
	for rows.Next() {
		var t models.ScheduledTimer
		err := rows.Scan(&t.ID, &t.UUID, &t.TaskID, &t.Title, &t.TriggerTime, &t.Status, &t.CreatedAt)
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

func (s *SQLiteStore) addTaskHistory(history *models.TaskHistory) error {
	stamp := any(nil)
	if !history.Timestamp.IsZero() {
		stamp = formatTime(history.Timestamp)
	}
	query := `INSERT INTO task_history (uuid, task_id, action, details, timestamp) VALUES (?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP))`
	history.UUID = newUUID()
	res, err := s.q.Exec(query, history.UUID, history.TaskID, history.Action, history.Details, stamp)
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	history.ID = id
	return s.stamp("task_history", history.UUID, "_created")
}

func (s *SQLiteStore) GetTaskHistory(taskID int64) ([]models.TaskHistory, error) {
	query := `SELECT id, uuid, task_id, action, details, timestamp FROM task_history WHERE task_id = ? ORDER BY id DESC`
	rows, err := s.q.Query(query, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var history []models.TaskHistory
	for rows.Next() {
		var h models.TaskHistory
		err := rows.Scan(&h.ID, &h.UUID, &h.TaskID, &h.Action, &h.Details, &h.Timestamp)
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

func whenKindOrDefault(w models.When) models.WhenKind {
	if w.Kind == "" {
		return models.WhenAnytime
	}
	return w.Kind
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// sortKeyTables are the tables with manual ordering.
var sortKeyTables = map[string]bool{"tasks": true, "projects": true, "areas": true, "headings": true}

// MaxSortKey returns the greatest sort key in the table, or "" when it has none.
func (s *SQLiteStore) MaxSortKey(table string) (string, error) {
	if !sortKeyTables[table] {
		return "", fmt.Errorf("table %q has no sort key", table)
	}
	var k sql.NullString
	err := s.q.QueryRow("SELECT MAX(sort_key) FROM " + table).Scan(&k)
	return k.String, err
}
