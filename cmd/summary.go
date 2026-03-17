package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/ui"
)

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Show daily summary",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.HeaderStyle.Render(" Zenith Daily Summary "))

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

		// Tasks Section
		fmt.Println(ui.TitleStyle.Render("Pending Tasks:"))
		hasTasks := false
		for _, t := range tasks {
			if t.Status != "done" {
				fmt.Printf(" • [%s] %s\n", ui.TaskStyle.Render(fmt.Sprintf("%d", t.ID)), t.Title)
				hasTasks = true
			}
		}
		if !hasTasks {
			fmt.Println("  No pending tasks! Enjoy your day.")
		}
		fmt.Println()

		// Habits Section
		fmt.Println(ui.TitleStyle.Render("Daily Habits:"))
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

func init() {
	rootCmd.AddCommand(summaryCmd)
}
