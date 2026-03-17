package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/ui"
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for tasks and habits",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		query := args[0]
		fmt.Printf("Searching for: %s\n\n", ui.TitleStyle.Render(query))

		// Search Tasks
		tasks, err := store.SearchTasks(query)
		if err != nil {
			return err
		}

		fmt.Println(ui.TitleStyle.Render("Tasks Found:"))
		if len(tasks) == 0 {
			fmt.Println("  No matching tasks.")
		} else {
			for _, t := range tasks {
				fmt.Printf(" • [%s] %s\n", ui.TaskStyle.Render(fmt.Sprintf("%d", t.ID)), t.Title)
			}
		}
		fmt.Println()

		// Search Habits
		habits, err := store.SearchHabits(query)
		if err != nil {
			return err
		}

		fmt.Println(ui.TitleStyle.Render("Habits Found:"))
		if len(habits) == 0 {
			fmt.Println("  No matching habits.")
		} else {
			for _, h := range habits {
				fmt.Printf(" • %s\n", ui.HabitStyle.Render(h.Name))
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(searchCmd)
}
