package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"
	"github.com/yalcinumutt/zenith-cli/internal/models"
	"github.com/yalcinumutt/zenith-cli/internal/ui"
)

var planCmd = &cobra.Command{
	Use:   "plan",
	Short: "Interactive daily planning",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(ui.HeaderStyle.Render(" Daily Planning "))

		tasks, err := app.Tasks.List()
		if err != nil {
			return err
		}

		fmt.Println("What would you like to focus on today?")
		fmt.Println("Select tasks by ID to add to today's plan (enter 'done' when finished):")

		for _, t := range tasks {
			if t.Status == models.StatusOpen && t.PlannedDate == nil {
				fmt.Printf("[%d] %s\n", t.ID, t.Title)
			}
		}

		var input string
		today := time.Now()
		for {
			fmt.Print("> ")
			fmt.Scanln(&input)
			if input == "done" || input == "" {
				break
			}

			id, err := strconv.ParseInt(input, 10, 64)
			if err != nil {
				fmt.Println("Invalid ID. Try again or type 'done'.")
				continue
			}

			for i := range tasks {
				if tasks[i].ID == id {
					if err := app.Tasks.Plan(id, today); err != nil {
						fmt.Printf("Error updating task %d: %v\n", id, err)
					} else {
						fmt.Printf("Added '%s' to today's plan!\n", tasks[i].Title)
					}
					break
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(planCmd)
}
