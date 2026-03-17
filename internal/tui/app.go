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
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "enter":
			if len(m.tasks) > 0 {
				task := &m.tasks[m.cursor]
				if task.Status == "done" {
					task.Status = "todo"
				} else {
					task.Status = "done"
				}
				_ = m.store.UpdateTask(task)
			}
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.tasks)-1 {
				m.cursor++
			}
		}
	}
	return m, nil
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

		s += fmt.Sprintf("%s%s %s\n", cursor, status, title)
	}

	s += "\n" + lipgloss.NewStyle().Foreground(ui.GrayColor).Render("(j/k: move, enter: toggle, q: quit)") + "\n"
	return ui.BorderStyle.Render(s)
}

func Start(s storage.Store) error {
	p := tea.NewProgram(NewModel(s))
	if _, err := p.Run(); err != nil {
		return err
	}
	return nil
}
