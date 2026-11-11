package storage

import "github.com/yalcinumutt/zenith-cli/internal/models"

var _ = models.StatusOpen

// Every synced write goes through write(): the row change and its field stamps
// commit in one transaction, whether or not the caller already opened one.

func (s *SQLiteStore) AddTask(task *models.Task) error {
	return s.write(func(st *SQLiteStore) error { return st.addTask(task) })
}

func (s *SQLiteStore) UpdateTask(task *models.Task) error {
	return s.write(func(st *SQLiteStore) error { return st.updateTask(task) })
}

func (s *SQLiteStore) DeleteTask(id int64) error {
	return s.write(func(st *SQLiteStore) error { return st.deleteTask(id) })
}

func (s *SQLiteStore) AddTag(tag *models.Tag) error {
	return s.write(func(st *SQLiteStore) error { return st.addTag(tag) })
}

func (s *SQLiteStore) UpdateTag(tag *models.Tag) error {
	return s.write(func(st *SQLiteStore) error { return st.updateTag(tag) })
}

func (s *SQLiteStore) AttachTagToTask(taskID, tagID int64) error {
	return s.write(func(st *SQLiteStore) error { return st.attachTagToTask(taskID, tagID) })
}

func (s *SQLiteStore) DetachTagFromTask(taskID, tagID int64) error {
	return s.write(func(st *SQLiteStore) error { return st.detachTagFromTask(taskID, tagID) })
}

func (s *SQLiteStore) AddHabit(habit *models.Habit) error {
	return s.write(func(st *SQLiteStore) error { return st.addHabit(habit) })
}

func (s *SQLiteStore) LogHabit(habitID int64) error {
	return s.write(func(st *SQLiteStore) error { return st.logHabit(habitID) })
}

func (s *SQLiteStore) AddProject(project *models.Project) error {
	return s.write(func(st *SQLiteStore) error { return st.addProject(project) })
}

func (s *SQLiteStore) AddTaskHistory(history *models.TaskHistory) error {
	return s.write(func(st *SQLiteStore) error { return st.addTaskHistory(history) })
}

func (s *SQLiteStore) UpdateProject(p *models.Project) error {
	return s.write(func(st *SQLiteStore) error { return st.updateProject(p) })
}

func (s *SQLiteStore) DeleteProject(id int64) error {
	return s.write(func(st *SQLiteStore) error { return st.deleteProject(id) })
}

func (s *SQLiteStore) AddArea(a *models.Area) error {
	return s.write(func(st *SQLiteStore) error { return st.addArea(a) })
}

func (s *SQLiteStore) UpdateArea(a *models.Area) error {
	return s.write(func(st *SQLiteStore) error { return st.updateArea(a) })
}

func (s *SQLiteStore) DeleteArea(id int64) error {
	return s.write(func(st *SQLiteStore) error { return st.deleteArea(id) })
}

func (s *SQLiteStore) AddHeading(h *models.Heading) error {
	return s.write(func(st *SQLiteStore) error { return st.addHeading(h) })
}

func (s *SQLiteStore) UpdateHeading(h *models.Heading) error {
	return s.write(func(st *SQLiteStore) error { return st.updateHeading(h) })
}

func (s *SQLiteStore) DeleteHeading(id int64) error {
	return s.write(func(st *SQLiteStore) error { return st.deleteHeading(id) })
}

func (s *SQLiteStore) AddRepeater(r *models.Repeater) error {
	return s.write(func(st *SQLiteStore) error { return st.addRepeater(r) })
}

func (s *SQLiteStore) UpdateRepeater(r *models.Repeater) error {
	return s.write(func(st *SQLiteStore) error { return st.updateRepeater(r) })
}

func (s *SQLiteStore) DeleteRepeater(id int64) error {
	return s.write(func(st *SQLiteStore) error { return st.deleteRepeater(id) })
}
