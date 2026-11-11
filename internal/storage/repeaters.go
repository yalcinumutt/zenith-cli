package storage

import (
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/yalcinumutt/zenith-cli/internal/models"
)

// Plain persistence for Repeaters. The rules (next date, generation) are in core.

const repeaterSelect = "id, uuid, kind, interval, weekdays, month_day, title, description, project_id, area_id, heading_id, priority, evening, deadline_offset, created_at, updated_at"

func scanRepeater(row interface{ Scan(...any) error }) (*models.Repeater, error) {
	var r models.Repeater
	var kind, weekdays string
	var offset sql.NullInt64
	err := row.Scan(&r.ID, &r.UUID, &kind, &r.Rule.Interval, &weekdays, &r.Rule.MonthDay, &r.Title, &r.Description,
		&r.ProjectID, &r.AreaID, &r.HeadingID, &r.Priority, &r.Evening, &offset, &r.CreatedAt, &r.UpdatedAt)
	if err != nil {
		return nil, err
	}
	r.Rule.Kind = models.RepeatKind(kind)
	for _, p := range strings.Split(weekdays, ",") {
		if p == "" {
			continue
		}
		if d, err := strconv.Atoi(p); err == nil {
			r.Rule.Weekdays = append(r.Rule.Weekdays, d)
		}
	}
	if offset.Valid {
		v := int(offset.Int64)
		r.DeadlineOffset = &v
	}
	return &r, nil
}

func weekdaysCSV(days []int) string {
	parts := make([]string, len(days))
	for i, d := range days {
		parts[i] = strconv.Itoa(d)
	}
	return strings.Join(parts, ",")
}

func offsetArg(r *models.Repeater) any {
	if r.DeadlineOffset == nil {
		return nil
	}
	return *r.DeadlineOffset
}

func (s *SQLiteStore) addRepeater(r *models.Repeater) error {
	r.UUID = newUUID()
	res, err := s.q.Exec(`INSERT INTO repeaters (uuid, kind, interval, weekdays, month_day, title, description, project_id, area_id, heading_id, priority, evening, deadline_offset)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.UUID, string(r.Rule.Kind), r.Rule.Interval, weekdaysCSV(r.Rule.Weekdays), r.Rule.MonthDay, r.Title, r.Description,
		r.ProjectID, r.AreaID, r.HeadingID, r.Priority, r.Evening, offsetArg(r))
	if err != nil {
		return err
	}
	if r.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	return s.stamp("repeater", r.UUID, repeaterCols...)
}

func (s *SQLiteStore) GetRepeater(id int64) (*models.Repeater, error) {
	r, err := scanRepeater(s.q.QueryRow("SELECT "+repeaterSelect+" FROM repeaters WHERE id = ? AND deleted_at IS NULL", id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	return r, err
}

func (s *SQLiteStore) updateRepeater(r *models.Repeater) error {
	if err := s.stampDiff("repeaters", "repeater", r.ID, repeaterCols, repeaterVals(r)); err != nil {
		return err
	}
	_, err := s.q.Exec(`UPDATE repeaters SET kind = ?, interval = ?, weekdays = ?, month_day = ?, title = ?, description = ?,
		project_id = ?, area_id = ?, heading_id = ?, priority = ?, evening = ?, deadline_offset = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?`,
		string(r.Rule.Kind), r.Rule.Interval, weekdaysCSV(r.Rule.Weekdays), r.Rule.MonthDay, r.Title, r.Description,
		r.ProjectID, r.AreaID, r.HeadingID, r.Priority, r.Evening, offsetArg(r), stamp(r.UpdatedAt), r.ID)
	return err
}

// DeleteRepeater tombstones the repeater. Tasks that point at it keep the
// reference; a tombstoned repeater reads as absent, which ends the series.
func (s *SQLiteStore) deleteRepeater(id int64) error {
	return s.softDelete("repeaters", "repeater", id)
}

// FindOccurrence returns the task generated for the repeater on the given day
// (any status), or ErrNotFound.
func (s *SQLiteStore) FindOccurrence(repeaterID int64, date string) (*models.Task, error) {
	tasks, err := s.queryTasks("t.repeater_id = ? AND t.occurrence_date = ?", taskOrder, repeaterID, date)
	if err != nil {
		return nil, err
	}
	if len(tasks) == 0 {
		return nil, ErrNotFound
	}
	return &tasks[0], nil
}
