package ai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAICompatibleRequest(t *testing.T) {
	var gotAuth, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath = r.Header.Get("Authorization"), r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		w.Write([]byte(`{"choices":[{"message":{"content":"hi"}}]}`))
	}))
	defer srv.Close()
	got, err := ProviderConfig{Kind: KindOpenAI, URL: srv.URL + "/v1/", Model: "m", APIKey: "sk-test"}.Complete(context.Background(), "sys", "usr")
	if err != nil || got != "hi" {
		t.Fatalf("%q %v", got, err)
	}
	if gotAuth != "Bearer sk-test" || gotPath != "/v1/chat/completions" || gotBody["model"] != "m" {
		t.Fatalf("auth=%q path=%q body=%v", gotAuth, gotPath, gotBody)
	}
}

func TestAnthropicRequest(t *testing.T) {
	var key, ver, path string
	var body map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key, ver, path = r.Header.Get("x-api-key"), r.Header.Get("anthropic-version"), r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &body)
		w.Write([]byte(`{"content":[{"type":"text","text":"a"},{"type":"text","text":"b"}]}`))
	}))
	defer srv.Close()
	got, err := ProviderConfig{Kind: KindAnthropic, URL: srv.URL, Model: "claude-x", APIKey: "k"}.Complete(context.Background(), "sys", "usr")
	if err != nil || got != "ab" {
		t.Fatalf("%q %v", got, err)
	}
	if key != "k" || ver == "" || path != "/messages" || body["system"] != "sys" || body["model"] != "claude-x" {
		t.Fatalf("key=%q ver=%q path=%q body=%v", key, ver, path, body)
	}
}

func TestProviderErrors(t *testing.T) {
	if _, err := (ProviderConfig{Kind: KindOpenAI}).Complete(context.Background(), "", ""); err == nil {
		t.Fatal("a cloud provider without a key is not ready")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "bad key", http.StatusUnauthorized) }))
	defer srv.Close()
	_, err := ProviderConfig{Kind: KindOpenAI, URL: srv.URL, APIKey: "x"}.Complete(context.Background(), "", "")
	if err == nil || !strings.Contains(err.Error(), "401") {
		t.Fatalf("%v", err)
	}
	if _, err := (ProviderConfig{Kind: KindLocal, URL: "http://127.0.0.1:1"}).Complete(context.Background(), "", ""); err == nil || !strings.Contains(err.Error(), "local AI") {
		t.Fatalf("%v", err)
	}
}

func TestParsePlanToleratesChatter(t *testing.T) {
	text := "Sure! Here is the plan:\n```json\n{\"summary\":\"Deep work first.\",\"blocks\":[{\"task_id\":3,\"start\":\"09:30\",\"duration\":45,\"reason\":\"due {today}\"},{\"task_id\":4,\"start\":\"later\",\"duration\":30}]}\n```\nEnjoy."
	sum, out, err := parsePlan(text)
	if err != nil {
		t.Fatal(err)
	}
	if sum != "Deep work first." || len(out) != 1 || out[0].TaskID != 3 || out[0].Start != 570 || out[0].Duration != 45 || out[0].Reason != "due {today}" {
		t.Fatalf("%q %+v", sum, out)
	}
	for _, bad := range []string{"no json here", "{broken", `{"blocks": "nope"}`} {
		if _, _, err := parsePlan(bad); err == nil {
			t.Errorf("parsePlan(%q) should fail", bad)
		}
	}
}

func TestPromptOmitsEverythingButSchedulingFacts(t *testing.T) {
	p := buildPrompt(PlanRequest{Date: "2026-03-01", WorkStart: 540, WorkEnd: 1020, EarliestM: 600,
		Tasks: []PlanTask{{ID: 7, Title: "Write report", Minutes: 60, Priority: 2, Deadline: "2026-03-02"}},
		Busy:  []Span{{Start: 720, End: 780}}})
	for _, want := range []string{"09:00-17:00", "10:00", "12:00-13:00", `id 7: "Write report", 60 minutes, priority 2 of 3, deadline 2026-03-02`} {
		if !strings.Contains(p, want) {
			t.Errorf("prompt missing %q:\n%s", want, p)
		}
	}
}
