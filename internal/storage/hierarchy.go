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
	return s.queryTasks("t.project_id = ?", taskOrder, projectID)
}

func (s *SQLiteStore) GetTasksInHeading(headingID int64) ([]models.Task, error) {
	return s.queryTasks("t.heading_id = ?", taskOrder, headingID)
}

func (s *SQLiteStore) GetTasksInArea(areaID int64) ([]models.Task, error) {
	return s.queryTasks("t.area_id = ?", taskOrder, areaID)
}

// ---- projects

func (s *SQLiteStore) GetProject(id int64) (*models.Project, error) {
	var p models.Project
	var sortKey sql.NullString
	err := s.q.QueryRow("SELECT id, uuid, area_id, name, description, status, sort_key, created_at, updated_at FROM projects WHERE id = ? AND deleted_at IS NULL", id).
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

func (s *SQLiteStore) updateProject(p *models.Project) error {
	if err := s.stampDiff("projects", "project", p.ID, projectCols, projectVals(p)); err != nil {
		return err
	}
	_, err := s.q.Exec("UPDATE projects SET area_id = ?, name = ?, description = ?, status = ?, sort_key = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?",
		p.AreaID, p.Name, p.Description, p.Status, nullIfEmpty(p.SortKey), stamp(p.UpdatedAt), p.ID)
	return err
}

// DeleteProject tombstones the project. Its headings and tasks are tombstoned
// explicitly by the core.
func (s *SQLiteStore) deleteProject(id int64) error { return s.softDelete("projects", "project", id) }

// ---- areas

func (s *SQLiteStore) addArea(a *models.Area) error {
	a.UUID = newUUID()
	res, err := s.q.Exec("INSERT INTO areas (uuid, name, sort_key) VALUES (?, ?, ?)", a.UUID, a.Name, nullIfEmpty(a.SortKey))
	if err != nil {
		return err
	}
	if a.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	return s.stamp("area", a.UUID, areaCols...)
}

func (s *SQLiteStore) GetAreas() ([]models.Area, error) {
	rows, err := s.q.Query("SELECT id, uuid, name, sort_key, created_at, updated_at FROM areas WHERE deleted_at IS NULL ORDER BY sort_key, uuid")
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
	err := s.q.QueryRow("SELECT id, uuid, name, sort_key, created_at, updated_at FROM areas WHERE id = ? AND deleted_at IS NULL", id).
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

func (s *SQLiteStore) updateArea(a *models.Area) error {
	if err := s.stampDiff("areas", "area", a.ID, areaCols, areaVals(a)); err != nil {
		return err
	}
	_, err := s.q.Exec("UPDATE areas SET name = ?, sort_key = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?", a.Name, nullIfEmpty(a.SortKey), stamp(a.UpdatedAt), a.ID)
	return err
}

func (s *SQLiteStore) deleteArea(id int64) error { return s.softDelete("areas", "area", id) }

// ---- headings

func (s *SQLiteStore) addHeading(h *models.Heading) error {
	h.UUID = newUUID()
	res, err := s.q.Exec("INSERT INTO headings (uuid, project_id, title, sort_key) VALUES (?, ?, ?, ?)", h.UUID, h.ProjectID, h.Title, nullIfEmpty(h.SortKey))
	if err != nil {
		return err
	}
	if h.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	return s.stamp("heading", h.UUID, headingCols...)
}

func (s *SQLiteStore) GetHeadings(projectID int64) ([]models.Heading, error) {
	rows, err := s.q.Query("SELECT id, uuid, project_id, title, sort_key, created_at, updated_at FROM headings WHERE project_id = ? AND deleted_at IS NULL ORDER BY sort_key, uuid", projectID)
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
	err := s.q.QueryRow("SELECT id, uuid, project_id, title, sort_key, created_at, updated_at FROM headings WHERE id = ? AND deleted_at IS NULL", id).
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

func (s *SQLiteStore) updateHeading(h *models.Heading) error {
	if err := s.stampDiff("headings", "heading", h.ID, headingCols, headingVals(h)); err != nil {
		return err
	}
	_, err := s.q.Exec("UPDATE headings SET project_id = ?, title = ?, sort_key = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?",
		h.ProjectID, h.Title, nullIfEmpty(h.SortKey), stamp(h.UpdatedAt), h.ID)
	return err
}

func (s *SQLiteStore) deleteHeading(id int64) error { return s.softDelete("headings", "heading", id) }

// stamp formats an injected time, or nil to let the database default apply.
func stamp(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return formatTime(t)
}
