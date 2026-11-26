package appapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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

type memSecrets map[string]string

func (m memSecrets) Get(kind string) (string, error) { return m[kind], nil }
func (m memSecrets) Set(kind, key string) error {
	if key == "" {
		delete(m, kind)
	} else {
		m[kind] = key
	}
	return nil
}

func icsServer(t *testing.T, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte(body)) }))
	t.Cleanup(srv.Close)
	return srv.URL
}

func TestCalendarFeedAndAutoPlan(t *testing.T) {
	s, _ := newService(t)
	s.secrets = memSecrets{}
	ics := "BEGIN:VCALENDAR\r\nBEGIN:VEVENT\r\nSUMMARY:Standup\r\nDTSTART:20260301T090000\r\nDTEND:20260301T100000\r\nEND:VEVENT\r\nEND:VCALENDAR\r\n"
	in := PlanSettingsDTO{WorkStart: 9 * 60, WorkEnd: 17 * 60, CalendarURL: icsServer(t, ics)}
	if r := s.TestCalendarURL(in.CalendarURL); !r.OK {
		t.Fatalf("a link can be tried before it is saved: %+v", r)
	}
	if r := s.TestCalendarURL("file:///etc/passwd"); r.OK {
		t.Fatal("a non-http link must be refused")
	}
	if _, err := s.SavePlanSettings(in); err != nil {
		t.Fatal(err)
	}
	if r := s.TestCalendar(); !r.OK || !strings.Contains(r.Message, "1 event") {
		t.Fatalf("test calendar: %+v", r)
	}
	task, _ := s.AddTask(NewTask{Title: "write", WhenKind: "scheduled", WhenDate: "2026-03-01"})
	est := 60
	if _, err := s.UpdateTask(task.ID, TaskPatch{Estimate: &est}); err != nil {
		t.Fatal(err)
	}

	p, err := s.Plan("2026-03-01")
	if err != nil {
		t.Fatal(err)
	}
	if !p.CalendarConfigured || len(p.Busy) != 1 || p.Busy[0].Title != "Standup" || p.Busy[0].Start != 540 || p.WorkStart != 540 {
		t.Fatalf("plan = %+v", p)
	}
	sg, err := s.SuggestPlan("2026-03-01", false)
	if err != nil {
		t.Fatal(err)
	}
	// The clock is 09:00; standup runs until 10:00, so the first free slot is 10:00.
	if len(sg.Proposals) != 1 || sg.Proposals[0].Start != 600 || sg.Proposals[0].TaskTitle != "write" || sg.Source != "heuristic" {
		t.Fatalf("suggestion = %+v", sg)
	}
	if blocks, _ := s.Plan("2026-03-01"); len(blocks.Blocks) != 0 {
		t.Fatal("suggesting must not save anything")
	}
	if err := s.ApplySuggestion("2026-03-01", sg.Proposals); err != nil {
		t.Fatal(err)
	}
	if p, _ = s.Plan("2026-03-01"); len(p.Blocks) != 1 || p.Blocks[0].Start != 600 {
		t.Fatalf("after apply = %+v", p.Blocks)
	}
	if again, _ := s.SuggestPlan("2026-03-01", false); len(again.Proposals) != 0 {
		t.Fatalf("a fully placed task needs no more time: %+v", again)
	}
}

func TestAISettingsKeyAndSuggestion(t *testing.T) {
	var gotAuth string
	var gotPrompt string
	llm := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		var req struct {
			Messages []struct{ Content string } `json:"messages"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		gotPrompt = req.Messages[len(req.Messages)-1].Content
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"summary\":\"Start with writing.\",\"blocks\":[{\"task_id\":1,\"start\":\"10:00\",\"duration\":60,\"reason\":\"fresh mind\"}]}"}}]}`))
	}))
	defer llm.Close()

	s, _ := newService(t)
	keys := memSecrets{}
	s.secrets = keys
	task, _ := s.AddTask(NewTask{Title: "SECRET-TITLE", WhenKind: "scheduled", WhenDate: "2026-03-01", Notes: "private notes"})
	est := 60
	_, _ = s.UpdateTask(task.ID, TaskPatch{Estimate: &est})

	if _, err := s.SuggestPlan("2026-03-01", true); err == nil {
		t.Fatal("AI suggestions need a provider")
	}
	st, err := s.SavePlanSettings(PlanSettingsDTO{WorkStart: 540, WorkEnd: 1020, AIKind: "openai", AIURL: llm.URL})
	if err != nil || st.AIHasKey {
		t.Fatalf("%+v %v", st, err)
	}
	if p, _ := s.Plan("2026-03-01"); p.AIReady {
		t.Fatal("a cloud provider without a key is not ready")
	}
	if err := s.SetAIKey("openai", "sk-test"); err != nil {
		t.Fatal(err)
	}
	if err := s.SetAIKey("local", "x"); err == nil {
		t.Fatal("a local provider has no key")
	}
	p, _ := s.Plan("2026-03-01")
	if !p.AIReady || !p.AICloud || !s.PlanSettings().AIHasKey {
		t.Fatalf("plan = %+v", p)
	}
	sg, err := s.SuggestPlan("2026-03-01", true)
	if err != nil {
		t.Fatal(err)
	}
	if sg.Source != "ai" || sg.Summary != "Start with writing." || len(sg.Proposals) != 1 || sg.Proposals[0].Start != 600 || sg.Proposals[0].Reason != "fresh mind" {
		t.Fatalf("%+v", sg)
	}
	if gotAuth != "Bearer sk-test" {
		t.Fatalf("auth = %q", gotAuth)
	}
	if !strings.Contains(gotPrompt, "SECRET-TITLE") || strings.Contains(gotPrompt, "private notes") {
		t.Fatalf("the prompt should carry titles but never notes:\n%s", gotPrompt)
	}
}

func TestSettingsValidation(t *testing.T) {
	s, _ := newService(t)
	bad := []PlanSettingsDTO{
		{WorkStart: 600, WorkEnd: 610},
		{WorkStart: 540, WorkEnd: 1020, AIKind: "skynet"},
		{WorkStart: 540, WorkEnd: 1020, CalendarURL: "file:///etc/passwd"},
		{WorkStart: 540, WorkEnd: 1020, AIKind: "local", AIURL: "javascript:alert(1)"},
	}
	for i, in := range bad {
		if _, err := s.SavePlanSettings(in); err == nil {
			t.Errorf("case %d should be rejected: %+v", i, in)
		}
	}
	got, err := s.SavePlanSettings(PlanSettingsDTO{WorkStart: 480, WorkEnd: 1020, CalendarURL: "webcal://example.test/a.ics"})
	if err != nil || got.CalendarURL != "https://example.test/a.ics" {
		t.Fatalf("%+v %v", got, err)
	}
	if again := s.PlanSettings(); again.WorkStart != 480 || again.CalendarURL != got.CalendarURL {
		t.Fatalf("settings should persist: %+v", again)
	}
}
