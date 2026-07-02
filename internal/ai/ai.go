package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/yalcinumut/zenith-cli/internal/models"
)

type Config struct {
	Provider string // ollama, lmstudio
	URL      string
	Model    string
}

func GetConfig() Config {
	provider := os.Getenv("ZENITH_AI_PROVIDER")
	if provider == "" {
		provider = "ollama"
	}

	url := os.Getenv("ZENITH_AI_URL")
	if url == "" {
		if provider == "lmstudio" {
			url = "http://localhost:1234/v1"
		} else {
			url = "http://localhost:11434/v1"
		}
	}

	model := os.Getenv("ZENITH_AI_MODEL")
	if model == "" {
		if provider == "lmstudio" {
			model = "meta-llama-3-8b-instruct" // Common default for LM Studio or matching
		} else {
			model = "llama3"
		}
	}

	return Config{
		Provider: provider,
		URL:      url,
		Model:    model,
	}
}

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model    string        `json:"model"`
	Messages []ChatMessage `json:"messages"`
}

type ChatResponse struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

func CallLLM(prompt string) (string, error) {
	cfg := GetConfig()
	apiURL := fmt.Sprintf("%s/chat/completions", cfg.URL)

	reqPayload := ChatRequest{
		Model: cfg.Model,
		Messages: []ChatMessage{
			{
				Role:    "system",
				Content: "You are Zenith AI, a helpful assistant built into the Zenith CLI productivity tool. Keep your responses concise, action-oriented, formatted in clean markdown, and highly motivational.",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqPayload)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("could not connect to AI Provider (%s) at %s. Please make sure your local LLM server is running.\nDetails: %w", cfg.Provider, cfg.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("AI Provider returned non-OK status: %s (Body: %s)", resp.Status, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("AI Provider returned empty choices list")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func SuggestTasks(tasks []models.Task, habits []models.Habit) (string, error) {
	var taskListStr string
	if len(tasks) == 0 {
		taskListStr = "- No current tasks.\n"
	} else {
		for _, t := range tasks {
			status := "Todo"
			if t.Status == "done" {
				status = "Completed"
			}
			taskListStr += fmt.Sprintf("- ID: %d, Title: %s, Status: %s, Priority: %d, Tracked Time: %ds\n", t.ID, t.Title, status, t.Priority, t.TotalTime)
		}
	}

	var habitListStr string
	if len(habits) == 0 {
		habitListStr = "- No current habits.\n"
	} else {
		for _, h := range habits {
			habitListStr += fmt.Sprintf("- Habit: %s, Frequency: %s\n", h.Name, h.Frequency)
		}
	}

	prompt := fmt.Sprintf(`Here are my current productivity items in Zenith:

Tasks:
%s
Habits:
%s

Please review my list and suggest:
1. Which 2-3 tasks I should prioritize today and why.
2. A suggested schedule or plan of action.
3. Quick motivational feedback on my habits.
Make your response punchy and formatted in beautiful terminal-friendly markdown.`, taskListStr, habitListStr)

	return CallLLM(prompt)
}

func GenerateSummary(tasks []models.Task, habits []models.Habit) (string, error) {
	var taskListStr string
	if len(tasks) == 0 {
		taskListStr = "- No tasks logged.\n"
	} else {
		for _, t := range tasks {
			status := "Todo"
			if t.Status == "done" {
				status = "Completed"
			}
			taskListStr += fmt.Sprintf("- ID: %d, Title: %s, Status: %s, Time Spent: %dh %dm %ds\n", t.ID, t.Title, status, t.TotalTime/3600, (t.TotalTime%3600)/60, t.TotalTime%60)
		}
	}

	prompt := fmt.Sprintf(`Analyze my progress and generate a daily summary review:

Tasks & Time Spent:
%s

Please output:
1. A summary of accomplishments (completed tasks and time spent).
2. Analysis of focus areas.
3. 2-3 constructive recommendations to improve my efficiency tomorrow.
Format the output in beautiful terminal-friendly markdown.`, taskListStr)

	return CallLLM(prompt)
}
