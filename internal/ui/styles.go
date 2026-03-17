package ui

import "github.com/charmbracelet/lipgloss"

var (
	// Colors
	PrimaryColor   = lipgloss.Color("#7D56F4")
	SecondaryColor = lipgloss.Color("#04B575")
	AccentColor    = lipgloss.Color("#EE6FF8")
	WhiteColor     = lipgloss.Color("#FFFFFF")
	GrayColor      = lipgloss.Color("#767676")

	// Styles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(WhiteColor).
			Background(PrimaryColor).
			Padding(0, 1).
			Bold(true).
			MarginBottom(1)

	TitleStyle = lipgloss.NewStyle().
			Foreground(PrimaryColor).
			Bold(true)

	HabitStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Italic(true)

	TaskStyle = lipgloss.NewStyle().
			Foreground(AccentColor)

	StatusDoneStyle = lipgloss.NewStyle().
			Foreground(SecondaryColor).
			Bold(true)

	StatusTodoStyle = lipgloss.NewStyle().
			Foreground(GrayColor)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(PrimaryColor).
			Padding(1, 2)
)
