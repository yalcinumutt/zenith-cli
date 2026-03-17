package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/models"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

var recurring string

var taskAddCmd = &cobra.Command{
	Use:   "add [title]",
	Short: "Add a new task",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		title := strings.Join(args, " ")
		task := &models.Task{
			Title:     title,
			Status:    "todo",
			Recurring: recurring,
		}
		if err := store.AddTask(task); err != nil {
			return err
		}
		fmt.Printf("Task added with ID: %d (Recurring: %s)\n", task.ID, task.Recurring)
		return nil
	},
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all tasks",
	RunE: func(cmd *cobra.Command, args []string) error {
		tasks, err := store.GetTasks()
		if err != nil {
			return err
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		fmt.Println("ID  | Status | Title")
		fmt.Println("----|--------|-------")
		for _, t := range tasks {
			fmt.Printf("%-3d | %-6s | %s\n", t.ID, t.Status, t.Title)
		}
		return nil
	},
}

var taskDoneCmd = &cobra.Command{
	Use:   "done [id]",
	Short: "Mark a task as done",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		tasks, err := store.GetTasks()
		if err != nil {
			return err
		}

		var target *models.Task
		for _, t := range tasks {
			if t.ID == id {
				target = &t
				break
			}
		}

		if target == nil {
			return fmt.Errorf("task with ID %d not found", id)
		}

		target.Status = "done"
		if err := store.UpdateTask(target); err != nil {
			return err
		}

		fmt.Printf("Task %d marked as done!\n", id)
		return nil
	},
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid ID: %w", err)
		}

		if err := store.DeleteTask(id); err != nil {
			return err
		}

		fmt.Printf("Task %d deleted.\n", id)
		return nil
	},
}

func init() {
	taskAddCmd.Flags().StringVarP(&recurring, "recurring", "r", "none", "Recurrence pattern (daily, weekly, monthly)")
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskDeleteCmd)
}
