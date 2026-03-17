package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/models"
	"github.com/yalcinumut/zenith-cli/internal/ui"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show daily summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.HeaderStyle.Render(" Zenith Daily Summary "))

		projects, err := store.GetProjects()
		if err != nil {
			return err
		}

		tasks, err := store.GetTasks()
		if err != nil {
			return err
		}

		habits, err := store.GetHabits()
		if err != nil {
			return err
		}

		now := time.Now().Format("Monday, 02 Jan 2006")
		fmt.Printf("Today is %s\n\n", ui.TitleStyle.Render(now))

		// Group tasks by project
		projectTasks := make(map[int64][]models.Task)
		for _, t := range tasks {
			pid := int64(0)
			if t.ProjectID != nil {
				pid = *t.ProjectID
			}
			projectTasks[pid] = append(projectTasks[pid], t)
		}

		// Tasks Section
		fmt.Println(ui.TitleStyle.Render("Project Tasks:"))
		
		// First show tasks without project
		if unassigned, ok := projectTasks[0]; ok {
			fmt.Println(ui.HeaderStyle.Copy().Background(ui.GrayColor).Render(" Unassigned "))
			renderTaskList(unassigned)
		}

		// Then show projects
		for _, p := range projects {
			if pts, ok := projectTasks[p.ID]; ok {
				fmt.Println(ui.HeaderStyle.Copy().Background(ui.AccentColor).Render(" " + p.Name + " "))
				renderTaskList(pts)
			}
		}

		// Habits Section
		fmt.Println("\n" + ui.TitleStyle.Render("Daily Habits:"))
		if len(habits) == 0 {
			fmt.Println("  No habits tracked yet.")
		} else {
			for _, h := range habits {
				fmt.Printf(" • %s (%s)\n", ui.HabitStyle.Render(h.Name), h.Frequency)
			}
		}

		return nil
	},
}

func renderTaskList(tasks []models.Task) {
	for _, t := range tasks {
		status := "[ ]"
		if t.Status == "done" {
			status = "[x]"
		}
		
		duration := fmt.Sprintf("%dh %dm", t.TotalTime/3600, (t.TotalTime%3600)/60)
		due := ""
		if t.DueDate != nil {
			due = " 📅 " + t.DueDate.Format("02 Jan")
		}

		timerIcon := ""
		if t.IsRunning {
			timerIcon = " ⏳"
		}

		fmt.Printf(" %s %s (%s)%s%s\n", status, t.Title, duration, due, timerIcon)
	}
}

func init() {
	rootCmd.AddCommand(summaryCmd)
}
