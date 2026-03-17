package tui

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/yalcinumut/zenith-cli/internal/models"
	"github.com/yalcinumut/zenith-cli/internal/storage"
	"github.com/yalcinumut/zenith-cli/internal/ui"
)

type model struct {
	store    storage.Store
	tasks    []models.Task
	cursor   int
	selected int
	err      error
}

func NewModel(s storage.Store) model {
	return model{
		store: s,
	}
}

func (m model) Init() tea.Cmd {
	return m.fetchTasks
}

func (m model) fetchTasks() tea.Msg {
	tasks, err := m.store.GetTasks()
	if err != nil {
		return err
	}
	return tasks
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []models.Task:
		m.tasks = msg
	case error:
		m.err = msg
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)
	}
	return m, nil
}

func (m model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "enter":
		return m.toggleTaskStatus()
	case "s":
		return m.toggleTaskTimer()
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(m.tasks)-1 {
			m.cursor++
		}
	}
	return m, nil
}

func (m model) toggleTaskStatus() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		return m, nil
	}
	task := &m.tasks[m.cursor]
	if task.Status == "done" {
		task.Status = "todo"
	} else {
		task.Status = "done"
		if task.IsRunning {
			_ = m.store.StopTaskTimer(task.ID)
		}
	}
	_ = m.store.UpdateTask(task)
	return m, m.fetchTasks
}

func (m model) toggleTaskTimer() (tea.Model, tea.Cmd) {
	if len(m.tasks) == 0 {
		return m, nil
	}
	task := m.tasks[m.cursor]
	if task.IsRunning {
		_ = m.store.StopTaskTimer(task.ID)
	} else {
		_ = m.store.StartTaskTimer(task.ID)
	}
	return m, m.fetchTasks
}

func (m model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v", m.err)
	}

	header := ui.HeaderStyle.Render(" Zenith TUI Dashboard ")
	s := header + "\n\n"

	for i, task := range m.tasks {
		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		status := ui.StatusTodoStyle.Render("[ ]")
		if task.Status == "done" {
			status = ui.StatusDoneStyle.Render("[x]")
		}

		title := task.Title
		if m.cursor == i {
			title = ui.TitleStyle.Render(title)
		}

		// Priority display
		priority := ""
		switch task.Priority {
		case models.PriorityHigh:
			priority = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#FFA500")).Render("!")
		case models.PriorityCritical:
			priority = " " + lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Bold(true).Render("!!")
		}

		// Tags display
		tagsStr := ""
		for _, tag := range task.Tags {
			tagStyle := lipgloss.NewStyle().Foreground(lipgloss.Color(tag.Color))
			if tag.Color == "" {
				tagStyle = lipgloss.NewStyle().Foreground(ui.AccentColor)
			}
			tagsStr += tagStyle.Render(" #" + tag.Name)
		}

		// Timer display
		duration := fmt.Sprintf("%dh %dm", task.TotalTime/3600, (task.TotalTime%3600)/60)
		timer := ui.StatusTodoStyle.Render(" (" + duration + ")")
		if task.IsRunning {
			timer = lipgloss.NewStyle().Foreground(ui.SecondaryColor).Bold(true).Render(" (" + duration + " ⏳)")
		}

		s += fmt.Sprintf("%s%s %s%s%s%s\n", cursor, status, title, priority, timer, tagsStr)
	}

	s += "\n" + lipgloss.NewStyle().Foreground(ui.GrayColor).Render("(j/k: move, enter: toggle, s: timer, q: quit)") + "\n"
	return ui.BorderStyle.Render(s)
}

func Start(s storage.Store) error {
	p := tea.NewProgram(NewModel(s))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
