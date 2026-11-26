package storage

import (
	"database/sql"
	"errors"

	"github.com/yalcinumutt/zenith-cli/internal/models"
)

const timeBlockSelect = `SELECT b.id, b.uuid, b.task_id, b.date, b.start_min, b.duration_min, b.created_at, b.updated_at
	FROM time_blocks b JOIN tasks t ON t.id = b.task_id AND t.deleted_at IS NULL
	WHERE b.deleted_at IS NULL`

func (s *SQLiteStore) addTimeBlock(b *models.TimeBlock) error {
	b.UUID = newUUID()
	res, err := s.q.Exec("INSERT INTO time_blocks (uuid, task_id, date, start_min, duration_min) VALUES (?, ?, ?, ?, ?)",
		b.UUID, b.TaskID, b.Date, b.StartMin, b.DurationMin)
	if err != nil {
		return err
	}
	if b.ID, err = res.LastInsertId(); err != nil {
		return err
	}
	return s.stamp("time_block", b.UUID, timeBlockCols...)
}

func (s *SQLiteStore) GetTimeBlock(id int64) (*models.TimeBlock, error) {
	var b models.TimeBlock
	err := s.q.QueryRow(timeBlockSelect+" AND b.id = ?", id).
		Scan(&b.ID, &b.UUID, &b.TaskID, &b.Date, &b.StartMin, &b.DurationMin, &b.CreatedAt, &b.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (s *SQLiteStore) GetTimeBlocks(date string) ([]models.TimeBlock, error) {
	rows, err := s.q.Query(timeBlockSelect+" AND b.date = ? ORDER BY b.start_min, b.uuid", date)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.TimeBlock
	for rows.Next() {
		var b models.TimeBlock
		if err := rows.Scan(&b.ID, &b.UUID, &b.TaskID, &b.Date, &b.StartMin, &b.DurationMin, &b.CreatedAt, &b.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}

func (s *SQLiteStore) updateTimeBlock(b *models.TimeBlock) error {
	if err := s.stampDiff("time_blocks", "time_block", b.ID, timeBlockCols, timeBlockVals(b)); err != nil {
		return err
	}
	_, err := s.q.Exec("UPDATE time_blocks SET task_id = ?, date = ?, start_min = ?, duration_min = ?, updated_at = COALESCE(?, CURRENT_TIMESTAMP) WHERE id = ?",
		b.TaskID, b.Date, b.StartMin, b.DurationMin, stamp(b.UpdatedAt), b.ID)
	return err
}
