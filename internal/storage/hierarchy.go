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
	return s.queryTasks(" WHERE t.project_id = ? ORDER BY t.id", projectID)
}

func (s *SQLiteStore) GetTasksInHeading(headingID int64) ([]models.Task, error) {
	return s.queryTasks(" WHERE t.heading_id = ? ORDER BY t.id", headingID)
}

func (s *SQLiteStore) GetTasksInArea(areaID int64) ([]models.Task, error) {
	return s.queryTasks(" WHERE t.area_id = ? ORDER BY t.id", areaID)
}

// ---- projects

func (s *SQLiteStore) GetProject(id int64) (*models.Project, error) {
	var p models.Project
	err := s.q.QueryRow("SELECT id, uuid, area_id, name, description, status, created_at, updated_at FROM projects WHERE id = ?", id).
		Scan(&p.ID, &p.UUID, &p.AreaID, &p.Name, &p.Description, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (s *SQLiteStore) UpdateProject(p *models.Project) error {
	_, err := s.q.Exec("UPDATE projects SET area_id = ?, name = ?, description = ?, status = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?",
		p.AreaID, p.Name, p.Description, p.Status, stamp(p.UpdatedAt), p.ID)
	return err
}

func (s *SQLiteStore) DeleteProject(id int64) error {
	_, err := s.q.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

// ---- areas

func (s *SQLiteStore) AddArea(a *models.Area) error {
	a.UUID = newUUID()
	res, err := s.q.Exec("INSERT INTO areas (uuid, name) VALUES (?, ?)", a.UUID, a.Name)
	if err != nil {
		return err
	}
	a.ID, err = res.LastInsertId()
	return err
}

func (s *SQLiteStore) GetAreas() ([]models.Area, error) {
	rows, err := s.q.Query("SELECT id, uuid, name, created_at, updated_at FROM areas ORDER BY id")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Area
	for rows.Next() {
		var a models.Area
		if err := rows.Scan(&a.ID, &a.UUID, &a.Name, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetArea(id int64) (*models.Area, error) {
	var a models.Area
	err := s.q.QueryRow("SELECT id, uuid, name, created_at, updated_at FROM areas WHERE id = ?", id).
		Scan(&a.ID, &a.UUID, &a.Name, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *SQLiteStore) UpdateArea(a *models.Area) error {
	_, err := s.q.Exec("UPDATE areas SET name = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?", a.Name, stamp(a.UpdatedAt), a.ID)
	return err
}

func (s *SQLiteStore) DeleteArea(id int64) error {
	_, err := s.q.Exec("DELETE FROM areas WHERE id = ?", id)
	return err
}

// ---- headings

func (s *SQLiteStore) AddHeading(h *models.Heading) error {
	h.UUID = newUUID()
	res, err := s.q.Exec("INSERT INTO headings (uuid, project_id, title) VALUES (?, ?, ?)", h.UUID, h.ProjectID, h.Title)
	if err != nil {
		return err
	}
	h.ID, err = res.LastInsertId()
	return err
}

func (s *SQLiteStore) GetHeadings(projectID int64) ([]models.Heading, error) {
	rows, err := s.q.Query("SELECT id, uuid, project_id, title, created_at, updated_at FROM headings WHERE project_id = ? ORDER BY id", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.Heading
	for rows.Next() {
		var h models.Heading
		if err := rows.Scan(&h.ID, &h.UUID, &h.ProjectID, &h.Title, &h.CreatedAt, &h.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) GetHeading(id int64) (*models.Heading, error) {
	var h models.Heading
	err := s.q.QueryRow("SELECT id, uuid, project_id, title, created_at, updated_at FROM headings WHERE id = ?", id).
		Scan(&h.ID, &h.UUID, &h.ProjectID, &h.Title, &h.CreatedAt, &h.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &h, nil
}

func (s *SQLiteStore) UpdateHeading(h *models.Heading) error {
	_, err := s.q.Exec("UPDATE headings SET project_id = ?, title = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?",
		h.ProjectID, h.Title, stamp(h.UpdatedAt), h.ID)
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
