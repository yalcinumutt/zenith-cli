package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/models"
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
		in := core.AddTaskInput{
			Title:     title,
			Recurring: recurring,
			Priority:  models.PriorityMedium,
		}

		if projectID != 0 {
			in.ProjectID = &projectID
		}

		if dueStr != "" {
			t, err := time.Parse("2006-01-02", dueStr)
			if err != nil {
				return fmt.Errorf("invalid due date format (use YYYY-MM-DD): %w", err)
			}
			in.DueDate = &t
		}

		if priorityStr != "" {
			switch strings.ToLower(priorityStr) {
			case "low":
				in.Priority = models.PriorityLow
			case "medium":
				in.Priority = models.PriorityMedium
			case "high":
				in.Priority = models.PriorityHigh
			case "critical":
				in.Priority = models.PriorityCritical
			default:
				return fmt.Errorf("invalid priority: %s (use low, medium, high, critical)", priorityStr)
			}
		}

		task, err := app.Tasks.Add(in)
		if err != nil {
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
		tasks, err := app.Tasks.List()
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

		if err := app.Tasks.Complete(id); err != nil {
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

		if err := app.Tasks.StartTimer(id); err != nil {
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

		if err := app.Tasks.StopTimer(id); err != nil {
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
		if err := app.Tasks.Tag(id, tagName, tagColor); err != nil {
			return err
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

		if err := app.Tasks.Delete(id); err != nil {
			return err
		}

		fmt.Printf("Task %d deleted.\n", id)
		return nil
	},
}

var taskMoveCmd = &cobra.Command{
	Use:   "move [taskID] [projectID]",
	Short: "Move a task to another project",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		taskID, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %w", err)
		}

		projectID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid project ID: %w", err)
		}

		if err := app.Tasks.MoveToProject(taskID, projectID); err != nil {
			return err
		}

		fmt.Printf("Task %d moved to project %d\n", taskID, projectID)
		return nil
	},
}

var taskHistoryCmd = &cobra.Command{
	Use:   "history [taskID]",
	Short: "View event history and logged sessions for a task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf(errMsgInvalidID, err)
		}

		history, err := app.Tasks.History(id)
		if err != nil {
			return err
		}

		if len(history) == 0 {
			fmt.Println("No history logs found for this task.")
			return nil
		}

		fmt.Printf("\nEvent History for Task ID %d:\n", id)
		fmt.Println("--------------------------------------------------------------------------------")
		for _, h := range history {
			timeStr := h.Timestamp.Format("2006-01-02 15:04:05")
			fmt.Printf("[%s] %-15s - %s\n", timeStr, strings.ToUpper(h.Action), h.Details)
		}
		fmt.Println("--------------------------------------------------------------------------------")
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
	taskCmd.AddCommand(taskMoveCmd)
	taskCmd.AddCommand(taskHistoryCmd)
}
