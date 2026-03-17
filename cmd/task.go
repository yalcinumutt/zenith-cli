package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/models"
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks",
}

var recurring string
var tagColor string
var projectID int64
var dueStr string
var priorityStr string

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

		if projectID != 0 {
			task.ProjectID = &projectID
		}

		if dueStr != "" {
			t, err := time.Parse("2006-01-02", dueStr)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
			task.DueDate = &t
		}

		if priorityStr != "" {
			switch strings.ToLower(priorityStr) {
			case "low":
				task.Priority = models.PriorityLow
			case "medium":
				task.Priority = models.PriorityMedium
			case "high":
				task.Priority = models.PriorityHigh
			case "critical":
				task.Priority = models.PriorityCritical
			default:
				return fmt.Errorf("invalid priority: %s (use low, medium, high, critical)", priorityStr)
			}
		}

		if err := store.AddTask(task); err != nil {
			return err
		}
		fmt.Printf("Task added with ID: %d\n", task.ID)
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

		fmt.Printf("%-3s | %-8s | %-20s | %-15s | %s\n", "ID", "Status", "Title", "Time", "Tags")
		fmt.Println("----|----------|----------------------|-----------------|-------")
		for _, t := range tasks {
			status := t.Status
			if t.IsRunning {
				status = "RUNNING"
			}

			// Format duration
			duration := fmt.Sprintf("%dh %dm %ds", t.TotalTime/3600, (t.TotalTime%3600)/60, t.TotalTime%60)

			// Format tags
			tagNames := []string{}
			for _, tag := range t.Tags {
				tagNames = append(tagNames, tag.Name)
			}
			tagsStr := strings.Join(tagNames, ", ")

			fmt.Printf("%-3d | %-8s | %-20s | %-15s | %s\n", t.ID, status, t.Title, duration, tagsStr)
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
			return fmt.Errorf(errMsgInvalidID, err)
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

const errMsgInvalidID = "invalid ID: %w"

var taskStartCmd = &cobra.Command{
	Use:   "start [id]",
	Short: "Start timer for a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf(errMsgInvalidID, err)
		}

		if err := store.StartTaskTimer(id); err != nil {
			return err
		}

		fmt.Printf("Started timer for task %d\n", id)
		return nil
	},
}

var taskStopCmd = &cobra.Command{
	Use:   "stop [id]",
	Short: "Stop timer for a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf(errMsgInvalidID, err)
		}

		if err := store.StopTaskTimer(id); err != nil {
			return err
		}

		fmt.Printf("Stopped timer for task %d\n", id)
		return nil
	},
}

var taskTagCmd = &cobra.Command{
	Use:   "tag [id] [tag_name]",
	Short: "Add a tag to a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf(errMsgInvalidID, err)
		}

		tagName := args[1]
		tag := &models.Tag{
			Name:  tagName,
			Color: tagColor,
		}

		if err := store.AddTag(tag); err != nil {
			return fmt.Errorf("could not create tag: %w", err)
		}

		if err := store.AttachTagToTask(id, tag.ID); err != nil {
			return fmt.Errorf("could not attach tag: %w", err)
		}

		fmt.Printf("Added tag '%s' to task %d\n", tagName, id)
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
			return fmt.Errorf(errMsgInvalidID, err)
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
	taskAddCmd.Flags().Int64VarP(&projectID, "project", "p", 0, "Project ID")
	taskAddCmd.Flags().StringVarP(&dueStr, "due", "d", "", "Due date (YYYY-MM-DD)")
	taskAddCmd.Flags().StringVarP(&priorityStr, "priority", "P", "medium", "Priority (low, medium, high, critical)")
	taskTagCmd.Flags().StringVarP(&tagColor, "color", "c", "", "Color for the tag")

	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskStopCmd)
	taskCmd.AddCommand(taskTagCmd)
}
