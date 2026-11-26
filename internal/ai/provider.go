package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// Provider kinds for the planner. "local" is an OpenAI-compatible server on this
// machine (Ollama, LM Studio); the other two send prompts to a cloud API.
const (
	KindLocal     = "local"
	KindOpenAI    = "openai"
	KindAnthropic = "anthropic"
)

// ProviderConfig says where and how to ask a model. The zero value is "no AI".
type ProviderConfig struct {
	Kind   string
	URL    string // base URL; empty uses the kind's default
	Model  string // empty uses the kind's default
	APIKey string // cloud kinds only
}

// IsCloud reports whether prompts leave this machine.
func (c ProviderConfig) IsCloud() bool { return c.Kind == KindOpenAI || c.Kind == KindAnthropic }

// Defaults are applied by Resolve so the UI can show what an empty field means.
func Defaults(kind string) (url, model string) {
	switch kind {
	case KindOpenAI:
		return "https://api.openai.com/v1", "gpt-4o-mini"
	case KindAnthropic:
		return "https://api.anthropic.com/v1", "claude-sonnet-5-5"
	case KindLocal:
		env := GetConfig()
		return env.URL, env.Model
	}
	return "", ""
}

// Resolve fills empty URL and model with the kind's defaults.
func (c ProviderConfig) Resolve() ProviderConfig {
	u, m := Defaults(c.Kind)
	if c.URL == "" {
		c.URL = u
	}
	if c.Model == "" {
		c.Model = m
	}
	c.URL = strings.TrimRight(c.URL, "/")
	return c
}

// Ready reports whether the config can be used: a known kind, and a key for cloud kinds.
func (c ProviderConfig) Ready() bool {
	switch c.Kind {
	case KindLocal:
		return true
	case KindOpenAI, KindAnthropic:
		return c.APIKey != ""
	}
	return false
}

// Complete sends one system+user exchange and returns the model's text.
func (c ProviderConfig) Complete(ctx context.Context, system, user string) (string, error) {
	c = c.Resolve()
	if !c.Ready() {
		return "", errors.New("the AI provider is not set up")
	}
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if c.Kind == KindAnthropic {
		return c.anthropic(ctx, system, user)
	}
	return c.openAI(ctx, system, user)
}

func (c ProviderConfig) post(ctx context.Context, url string, headers map[string]string, payload any, out any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if c.Kind == KindLocal {
			return fmt.Errorf("could not reach the local AI at %s: is the server running?", c.URL)
		}
		cause := errors.Unwrap(err)
		if cause == nil {
			cause = errors.New("network error")
		}
		return fmt.Errorf("could not reach the AI provider: %w", cause)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("the AI provider answered %s: %s", resp.Status, snippet(data))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("could not read the AI provider's answer: %w", err)
	}
	return nil
}

func snippet(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 200 {
		s = s[:200] + "…"
	}
	return s
}

func (c ProviderConfig) openAI(ctx context.Context, system, user string) (string, error) {
	headers := map[string]string{}
	if c.APIKey != "" {
		headers["Authorization"] = "Bearer " + c.APIKey
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	err := c.post(ctx, c.URL+"/chat/completions", headers, map[string]any{
		"model": c.Model,
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
		"temperature": 0.2,
	}, &out)
	if err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("the AI provider returned no answer")
	}
	return out.Choices[0].Message.Content, nil
}

func (c ProviderConfig) anthropic(ctx context.Context, system, user string) (string, error) {
	var out struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	err := c.post(ctx, c.URL+"/messages", map[string]string{"x-api-key": c.APIKey, "anthropic-version": "2023-06-01"}, map[string]any{
		"model":      c.Model,
		"max_tokens": 2048,
		"system":     system,
		"messages":   []map[string]string{{"role": "user", "content": user}},
	}, &out)
	if err != nil {
		return "", err
	}
	var sb strings.Builder
	for _, b := range out.Content {
		if b.Type == "text" {
			sb.WriteString(b.Text)
		}
	}
	if sb.Len() == 0 {
		return "", errors.New("the AI provider returned no answer")
	}
	return sb.String(), nil
}
