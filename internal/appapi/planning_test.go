package appapi

import "testing"

func TestPlanRoundTrip(t *testing.T) {
	s, _ := newService(t)
	task, err := s.AddTask(NewTask{Title: "write", WhenKind: "scheduled", WhenDate: "2026-03-01"})
	if err != nil {
		t.Fatal(err)
	}
	est := 45
	if d, err := s.UpdateTask(task.ID, TaskPatch{Estimate: &est}); err != nil || d.Estimate != 45 {
		t.Fatalf("estimate = %+v, %v", d, err)
	}
	blk, err := s.AddBlock(task.ID, "2026-03-01", 600, 45)
	if err != nil {
		t.Fatal(err)
	}
	p, err := s.Plan("") // empty means today
	if err != nil {
		t.Fatal(err)
	}
	if p.Date != "2026-03-01" || len(p.Tasks) != 1 || len(p.Blocks) != 1 || p.Blocks[0].TaskTitle != "write" || p.Blocks[0].Start != 600 {
		t.Fatalf("plan = %+v", p)
	}
	if err := s.ResizeBlock(blk.ID, 60); err != nil {
		t.Fatal(err)
	}
	if err := s.Complete(task.ID); err != nil {
		t.Fatal(err)
	}
	p, _ = s.Plan("2026-03-01")
	if len(p.Blocks) != 1 || !p.Blocks[0].Done || p.Blocks[0].Duration != 60 {
		t.Fatalf("after complete = %+v", p.Blocks)
	}
	if err := s.RemoveBlock(blk.ID); err != nil {
		t.Fatal(err)
	}
	if p, _ = s.Plan("2026-03-01"); len(p.Blocks) != 0 {
		t.Fatalf("after remove = %+v", p.Blocks)
	}
}
