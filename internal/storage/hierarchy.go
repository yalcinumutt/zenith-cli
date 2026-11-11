package storage

import (
	"database/sql"
	"errors"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
)

// Plain persistence for Area > Project > Heading containers. Rules (what may
// contain what, what deleting does) live in internal/core.

func (s *SQLiteStore) GetTasksInProject(projectID int64) ([]models.Task, error) {
	return s.queryTasks(" WHERE t.project_id = ?"+taskOrder, projectID)
}

func (s *SQLiteStore) GetTasksInHeading(headingID int64) ([]models.Task, error) {
	return s.queryTasks(" WHERE t.heading_id = ?"+taskOrder, headingID)
}

func (s *SQLiteStore) GetTasksInArea(areaID int64) ([]models.Task, error) {
	return s.queryTasks(" WHERE t.area_id = ?"+taskOrder, areaID)
}

// ---- projects

func (s *SQLiteStore) GetProject(id int64) (*models.Project, error) {
	var p models.Project
	var sortKey sql.NullString
	err := s.q.QueryRow("SELECT id, uuid, area_id, name, description, status, sort_key, created_at, updated_at FROM projects WHERE id = ?", id).
		Scan(&p.ID, &p.UUID, &p.AreaID, &p.Name, &p.Description, &p.Status, &sortKey, &p.CreatedAt, &p.UpdatedAt)
	p.SortKey = sortKey.String
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *SQLiteStore) UpdateProject(p *models.Project) error {
	_, err := s.q.Exec("UPDATE projects SET area_id = ?, name = ?, description = ?, status = ?, sort_key = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?",
		p.AreaID, p.Name, p.Description, p.Status, nullIfEmpty(p.SortKey), stamp(p.UpdatedAt), p.ID)
	return err
}

func (s *SQLiteStore) DeleteProject(id int64) error {
	_, err := s.q.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

// ---- areas

func (s *SQLiteStore) AddArea(a *models.Area) error {
	a.UUID = newUUID()
	res, err := s.q.Exec("INSERT INTO areas (uuid, name, sort_key) VALUES (?, ?, ?)", a.UUID, a.Name, nullIfEmpty(a.SortKey))
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	return err
}

func (s *SQLiteStore) GetAreas() ([]models.Area, error) {
	rows, err := s.q.Query("SELECT id, uuid, name, sort_key, created_at, updated_at FROM areas ORDER BY sort_key, uuid")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Area
	for rows.Next() {
		var a models.Area
		var sortKey sql.NullString
		if err := rows.Scan(&a.ID, &a.UUID, &a.Name, &sortKey, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		a.SortKey = sortKey.String
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetArea(id int64) (*models.Area, error) {
	var a models.Area
	var sortKey sql.NullString
	err := s.q.QueryRow("SELECT id, uuid, name, sort_key, created_at, updated_at FROM areas WHERE id = ?", id).
		Scan(&a.ID, &a.UUID, &a.Name, &sortKey, &a.CreatedAt, &a.UpdatedAt)
	a.SortKey = sortKey.String
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *SQLiteStore) UpdateArea(a *models.Area) error {
	_, err := s.q.Exec("UPDATE areas SET name = ?, sort_key = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?", a.Name, nullIfEmpty(a.SortKey), stamp(a.UpdatedAt), a.ID)
	return err
}

func (s *SQLiteStore) DeleteArea(id int64) error {
	_, err := s.q.Exec("DELETE FROM areas WHERE id = ?", id)
	return err
}

// ---- headings

func (s *SQLiteStore) AddHeading(h *models.Heading) error {
	h.UUID = newUUID()
	res, err := s.q.Exec("INSERT INTO headings (uuid, project_id, title, sort_key) VALUES (?, ?, ?, ?)", h.UUID, h.ProjectID, h.Title, nullIfEmpty(h.SortKey))
	if err != nil {
		return err
	}
	h.ID, err = res.LastInsertId()
	return err
}

func (s *SQLiteStore) GetHeadings(projectID int64) ([]models.Heading, error) {
	rows, err := s.q.Query("SELECT id, uuid, project_id, title, sort_key, created_at, updated_at FROM headings WHERE project_id = ? ORDER BY sort_key, uuid", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Heading
	for rows.Next() {
		var h models.Heading
		var sortKey sql.NullString
		if err := rows.Scan(&h.ID, &h.UUID, &h.ProjectID, &h.Title, &sortKey, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		h.SortKey = sortKey.String
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetHeading(id int64) (*models.Heading, error) {
	var h models.Heading
	var sortKey sql.NullString
	err := s.q.QueryRow("SELECT id, uuid, project_id, title, sort_key, created_at, updated_at FROM headings WHERE id = ?", id).
		Scan(&h.ID, &h.UUID, &h.ProjectID, &h.Title, &sortKey, &h.CreatedAt, &h.UpdatedAt)
	h.SortKey = sortKey.String
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (s *SQLiteStore) UpdateHeading(h *models.Heading) error {
	_, err := s.q.Exec("UPDATE headings SET project_id = ?, title = ?, sort_key = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?",
		h.ProjectID, h.Title, nullIfEmpty(h.SortKey), stamp(h.UpdatedAt), h.ID)
	return err
}

func (s *SQLiteStore) DeleteHeading(id int64) error {
	_, err := s.q.Exec("DELETE FROM headings WHERE id = ?", id)
	return err
}

// stamp formats an injected time, or nil to let the database default apply.
func stamp(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return formatTime(t)
}
