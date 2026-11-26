package appapi

import "github.com/yalcinumutt/zenith-cli/internal/models"

// ---- daily planning

// Plan returns a day's tasks and time blocks. An empty date means today.
func (s *Service) Plan(date string) (PlanDTO, error) {
	if date == "" {
		date = s.today()
	}
	p, err := s.core.Planning.Plan(date)
	if err != nil {
		return PlanDTO{}, err
	}
	tasks, err := s.toDTOs(p.Tasks)
	if err != nil {
		return PlanDTO{}, err
	}
	// A block can belong to a task that is not in the day's list (completed
	// since, or scheduled elsewhere); resolve its title individually.
	listed := map[int64]models.Task{}
	for _, t := range p.Tasks {
		listed[t.ID] = t
	}
	blocks := make([]BlockDTO, 0, len(p.Blocks))
	for _, b := range p.Blocks {
		t, ok := listed[b.TaskID]
		if !ok {
			full, err := s.core.Tasks.Get(b.TaskID)
			if err != nil {
				continue // task vanished between the two reads
			}
			t = *full
		}
		blocks = append(blocks, BlockDTO{
			ID: b.ID, UUID: b.UUID, TaskID: b.TaskID, TaskTitle: t.Title, Done: t.Status != models.StatusOpen,
			Date: b.Date, Start: b.StartMin, Duration: b.DurationMin,
		})
	}
	return PlanDTO{Date: date, Tasks: tasks, Blocks: blocks}, nil
}

func (s *Service) AddBlock(taskID int64, date string, start, duration int) (BlockDTO, error) {
	b, err := s.core.Planning.AddBlock(taskID, date, start, duration)
	if err != nil {
		return BlockDTO{}, err
	}
	t, err := s.core.Tasks.Get(taskID)
	if err != nil {
		return BlockDTO{}, err
	}
	return BlockDTO{ID: b.ID, UUID: b.UUID, TaskID: taskID, TaskTitle: t.Title, Date: b.Date, Start: b.StartMin, Duration: b.DurationMin}, nil
}

func (s *Service) MoveBlock(id int64, date string, start int) error {
	return s.core.Planning.Move(id, date, start)
}

func (s *Service) ResizeBlock(id int64, duration int) error {
	return s.core.Planning.Resize(id, duration)
}
func (s *Service) RemoveBlock(id int64) error { return s.core.Planning.Remove(id) }
