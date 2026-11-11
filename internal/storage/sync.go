package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/yalcinumutt/zenith-cli/internal/models"
)

// Sync groundwork (ADR 0010, ADR 0011). Every mutation of a synced entity
// records per-field stamps in the sidecar field_versions table in the same
// transaction as the change, and deletes are stamped soft deletes. No folder
// exchange ships yet; this makes adding it a pure feature, not a data migration.
//
// Device-local and unstamped: scheduled reminders, the running time log, and the
// sync_state row (device id and clock).

// syncCtx is shared by a store and the transaction-bound stores derived from it.
type syncCtx struct {
	now func() time.Time
}

// SetClock sets the wall clock used for hybrid-logical-clock stamps and
// tombstone times. The core wires its own clock in so tests can control it.
func (s *SQLiteStore) SetClock(now func() time.Time) { s.sync.now = now }

// write runs fn inside a transaction, joining the caller's when there is one, so
// a row change and its stamps always commit (or fail) together.
func (s *SQLiteStore) write(fn func(*SQLiteStore) error) error {
	if s.tx {
		return fn(s)
	}
	return s.WithTx(func(st Store) error { return fn(st.(*SQLiteStore)) })
}

// nextHLC advances the device's hybrid logical clock and returns a sortable
// stamp plus the device id. Stamps from one device are strictly increasing even
// if the wall clock stands still or steps backwards.
func (s *SQLiteStore) nextHLC() (hlc, device string, err error) {
	var ms, counter int64
	err = s.q.QueryRow("SELECT device_id, hlc_ms, hlc_counter FROM sync_state WHERE id = 1").Scan(&device, &ms, &counter)
	if errors.Is(err, sql.ErrNoRows) {
		device = newUUID()
		if _, err = s.q.Exec("INSERT INTO sync_state (id, device_id, hlc_ms, hlc_counter) VALUES (1, ?, 0, 0)", device); err != nil {
			return "", "", err
		}
	} else if err != nil {
		return "", "", err
	}
	now := s.sync.now().UnixMilli()
	if now > ms {
		ms, counter = now, 0
	} else {
		counter++
	}
	if _, err = s.q.Exec("UPDATE sync_state SET hlc_ms = ?, hlc_counter = ? WHERE id = 1", ms, counter); err != nil {
		return "", "", err
	}
	return fmt.Sprintf("%015d-%05d", ms, counter), device, nil
}

// stamp records that the fields of an entity were written now.
func (s *SQLiteStore) stamp(entity, uuid string, fields ...string) error {
	if len(fields) == 0 || uuid == "" {
		return nil
	}
	hlc, device, err := s.nextHLC()
	if err != nil {
		return err
	}
	for _, f := range fields {
		if _, err := s.q.Exec(`INSERT INTO field_versions (entity, uuid, field, hlc, device_id) VALUES (?, ?, ?, ?, ?)
			ON CONFLICT(entity, uuid, field) DO UPDATE SET hlc = excluded.hlc, device_id = excluded.device_id`,
			entity, uuid, f, hlc, device); err != nil {
			return err
		}
	}
	return nil
}

// rowFields reads the stored text of the given columns for diffing.
func (s *SQLiteStore) rowFields(table string, id int64, cols []string) (uuid string, fields map[string]string, ok bool, err error) {
	vals := make([]sql.NullString, len(cols)+1)
	ptrs := make([]any, len(vals))
	for i := range vals {
		ptrs[i] = &vals[i]
	}
	q := "SELECT uuid"
	for _, c := range cols {
		q += ", " + c
	}
	err = s.q.QueryRow(q+" FROM "+table+" WHERE id = ?", id).Scan(ptrs...)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil, false, nil
	}
	if err != nil {
		return "", nil, false, err
	}
	fields = map[string]string{}
	for i, c := range cols {
		fields[c] = vals[i+1].String
	}
	return vals[0].String, fields, true, nil
}

// stampDiff stamps only the fields whose stored value differs from nv. Call it
// before the UPDATE so the old values are still readable.
func (s *SQLiteStore) stampDiff(table, entity string, id int64, cols []string, nv map[string]string) error {
	uuid, old, ok, err := s.rowFields(table, id, cols)
	if err != nil || !ok {
		return err
	}
	var changed []string
	for _, c := range cols {
		if old[c] != nv[c] {
			changed = append(changed, c)
		}
	}
	return s.stamp(entity, uuid, changed...)
}

// softDelete tombstones a row (a stamped deleted_at); synced tables are never
// hard-deleted. Deleting an already-deleted row does nothing.
func (s *SQLiteStore) softDelete(table, entity string, id int64) error {
	var uuid string
	err := s.q.QueryRow("SELECT uuid FROM "+table+" WHERE id = ? AND deleted_at IS NULL", id).Scan(&uuid)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if _, err := s.q.Exec("UPDATE "+table+" SET deleted_at = ? WHERE id = ?", formatTime(s.sync.now()), id); err != nil {
		return err
	}
	return s.stamp(entity, uuid, "deleted_at")
}

// str renders a column value the way rowFields reads it back, so unchanged
// fields compare equal.
func str(v any) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case int:
		return strconv.Itoa(x)
	case int64:
		return strconv.FormatInt(x, 10)
	case *int64:
		if x == nil {
			return ""
		}
		return strconv.FormatInt(*x, 10)
	case *int:
		if x == nil {
			return ""
		}
		return strconv.Itoa(*x)
	case bool:
		if x {
			return "1"
		}
		return "0"
	case models.Priority:
		return strconv.Itoa(int(x))
	}
	return fmt.Sprint(v)
}

// Stamped field sets, by entity. Foreign references are listed by their local
// column; the op-log boundary translates them to uuids (ADR 0003).
var (
	taskCols     = []string{"project_id", "area_id", "heading_id", "parent_task_id", "title", "description", "status", "priority", "when_kind", "when_date", "deadline", "evening", "repeater_id", "occurrence_date", "sort_key"}
	projectCols  = []string{"area_id", "name", "description", "status", "sort_key"}
	areaCols     = []string{"name", "sort_key"}
	headingCols  = []string{"project_id", "title", "sort_key"}
	tagCols      = []string{"name", "color"}
	habitCols    = []string{"name", "description", "frequency"}
	repeaterCols = []string{"kind", "interval", "weekdays", "month_day", "title", "description", "project_id", "area_id", "heading_id", "priority", "evening", "deadline_offset"}
)

func taskVals(t *models.Task) map[string]string {
	return map[string]string{
		"project_id": str(t.ProjectID), "area_id": str(t.AreaID), "heading_id": str(t.HeadingID), "parent_task_id": str(t.ParentID),
		"title": t.Title, "description": t.Description, "status": t.Status, "priority": str(t.Priority),
		"when_kind": string(whenKindOrDefault(t.When)), "when_date": t.When.Date, "deadline": t.Deadline, "evening": str(t.Evening),
		"repeater_id": str(t.RepeaterID), "occurrence_date": t.OccurrenceDate, "sort_key": t.SortKey,
	}
}

func projectVals(p *models.Project) map[string]string {
	return map[string]string{"area_id": str(p.AreaID), "name": p.Name, "description": p.Description, "status": p.Status, "sort_key": p.SortKey}
}

func areaVals(a *models.Area) map[string]string {
	return map[string]string{"name": a.Name, "sort_key": a.SortKey}
}

func headingVals(h *models.Heading) map[string]string {
	return map[string]string{"project_id": str(h.ProjectID), "title": h.Title, "sort_key": h.SortKey}
}

func tagVals(t *models.Tag) map[string]string {
	return map[string]string{"name": t.Name, "color": t.Color}
}

func habitVals(h *models.Habit) map[string]string {
	return map[string]string{"name": h.Name, "description": h.Description, "frequency": h.Frequency}
}

func repeaterVals(r *models.Repeater) map[string]string {
	return map[string]string{
		"kind": string(r.Rule.Kind), "interval": str(r.Rule.Interval), "weekdays": weekdaysCSV(r.Rule.Weekdays), "month_day": str(r.Rule.MonthDay),
		"title": r.Title, "description": r.Description, "project_id": str(r.ProjectID), "area_id": str(r.AreaID), "heading_id": str(r.HeadingID),
		"priority": str(r.Priority), "evening": str(r.Evening), "deadline_offset": str(r.DeadlineOffset),
	}
}
