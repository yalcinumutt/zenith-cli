package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/ui"
)

var logCmd = &cobra.Command{
	Use:   "log",
	Short: "Show historical activity log",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.HeaderStyle.Render(" Zenith Activity Log "))

		tasks, err := store.GetTasks()
		if err != nil {
			return err
		}

		doneTasks := []string{}
		for _, t := range tasks {
			if t.Status == "done" {
				duration := fmt.Sprintf("%dh %dm %ds", t.TotalTime/3600, (t.TotalTime%3600)/60, t.TotalTime%60)
				doneTasks = append(doneTasks, fmt.Sprintf(" • %s (Time: %s)", t.Title, duration))
			}
		}

		if len(doneTasks) == 0 {
			fmt.Println("No tasks finished yet. Time to get to work!")
		} else {
			fmt.Println(ui.TitleStyle.Render("Recently Finished:"))
			fmt.Println(strings.Join(doneTasks, "\n"))
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(logCmd)
}
