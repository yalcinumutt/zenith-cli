package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/models"
)

var habitCmd = &cobra.Command{
	Use:   "habit",
	Short: "Manage habits",
}

var habitAddCmd = &cobra.Command{
	Use:   "add [name]",
	Short: "Add a new habit",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := strings.Join(args, " ")
		habit := &models.Habit{
			Name:      name,
			Frequency: "daily",
		}
		if err := store.AddHabit(habit); err != nil {
			return err
		}
		fmt.Printf("Habit added with ID: %d\n", habit.ID)
		return nil
	},
}

var habitListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all habits",
	RunE: func(cmd *cobra.Command, args []string) error {
		habits, err := store.GetHabits()
		if err != nil {
			return err
		}

		if len(habits) == 0 {
			fmt.Println("No habits found.")
			return nil
		}

		fmt.Println("ID  | Frequency | Name")
		fmt.Println("----|-----------|------")
		for _, h := range habits {
			fmt.Printf("%-3d | %-9s | %s\n", h.ID, h.Frequency, h.Name)
		}
		return nil
	},
}

var habitLogCmd = &cobra.Command{
	Use:   "log [id]",
	Short: "Log a habit completion",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		if err := store.LogHabit(id); err != nil {
			return err
		}

		fmt.Printf("Habit %d logged!\n", id)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(habitCmd)
	habitCmd.AddCommand(habitAddCmd)
	habitCmd.AddCommand(habitListCmd)
	habitCmd.AddCommand(habitLogCmd)
}
