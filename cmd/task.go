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
var parentID int64
var areaFlag int64
var headingFlag int64
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
		if parentID != 0 {
			in.ParentID = &parentID
		}
		if areaFlag != 0 {
			in.AreaID = &areaFlag
		}
		if headingFlag != 0 {
			in.HeadingID = &headingFlag
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

		fmt.Printf("%-3s | %-9s | %-20s | %-15s | %s\n", "ID", "Status", "Title", "Time", "Tags")
		fmt.Println("----|-----------|----------------------|-----------------|-------")
		printTaskTree(tasks)
		return nil
	},
}

// printTaskTree prints top-level tasks, each followed by its Subtasks nested
// beneath it. A Subtask whose parent is missing is shown at the top level.
func printTaskTree(tasks []models.Task) {
	byID := map[int64]bool{}
	children := map[int64][]models.Task{}
	for _, t := range tasks {
		byID[t.ID] = true
	}
	var roots []models.Task
	for _, t := range tasks {
		if t.ParentID != nil && byID[*t.ParentID] {
			children[*t.ParentID] = append(children[*t.ParentID], t)
		} else {
			roots = append(roots, t)
		}
	}
	printed := map[int64]bool{}
	var walk func(t models.Task, depth int)
	walk = func(t models.Task, depth int) {
		if printed[t.ID] {
			return
		}
		printed[t.ID] = true
		printTaskRow(t, depth)
		for _, c := range children[t.ID] {
			walk(c, depth+1)
		}
	}
	for _, t := range roots {
		walk(t, 0)
	}
}

func printTaskRow(t models.Task, depth int) {
	status := t.Status
	if t.InProgress() {
		status = "RUNNING"
	}

	duration := fmt.Sprintf("%dh %dm %ds", t.TotalTime/3600, (t.TotalTime%3600)/60, t.TotalTime%60)

	tagNames := []string{}
	for _, tag := range t.Tags {
		tagNames = append(tagNames, tag.Name)
	}

	title := t.Title
	if depth > 0 {
		title = strings.Repeat("  ", depth-1) + "└ " + title
	}
	fmt.Printf("%-3d | %-9s | %-20s | %-15s | %s\n", t.ID, status, title, duration, strings.Join(tagNames, ", "))
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

var taskCancelCmd = &cobra.Command{
	Use:   "cancel [id]",
	Short: "Cancel a task and its open subtasks",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf(errMsgInvalidID, err)
		}
		if err := app.Tasks.Cancel(id); err != nil {
			return err
		}
		fmt.Printf("Task %d cancelled.\n", id)
		return nil
	},
}

var taskReopenCmd = &cobra.Command{
	Use:   "reopen [id]",
	Short: "Reopen a completed or cancelled task",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf(errMsgInvalidID, err)
		}
		if err := app.Tasks.Reopen(id); err != nil {
			return err
		}
		fmt.Printf("Task %d reopened.\n", id)
		return nil
	},
}

var taskParentCmd = &cobra.Command{
	Use:   "parent [id] [parentID|none]",
	Short: "Make a task a subtask of another task, or top-level with 'none'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf(errMsgInvalidID, err)
		}
		var parent *int64
		if args[1] != "none" {
			p, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid parent ID: %w", err)
			}
			parent = &p
		}
		if err := app.Tasks.SetParent(id, parent); err != nil {
			return err
		}
		if parent == nil {
			fmt.Printf("Task %d is now top-level.\n", id)
		} else {
			fmt.Printf("Task %d is now a subtask of %d.\n", id, *parent)
		}
		return nil
	},
}

var taskAreaCmd = &cobra.Command{
	Use:   "area [id] [areaID|none]",
	Short: "File a task directly in an area, or unfile it with 'none'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("task ID", args[0])
		if err != nil {
			return err
		}
		areaID, err := parseOptionalID("area ID", args[1])
		if err != nil {
			return err
		}
		if areaID == nil {
			err = app.Tasks.Unfile(id)
		} else {
			err = app.Tasks.FileInArea(id, *areaID)
		}
		if err != nil {
			return err
		}
		fmt.Printf("Task %d filed.\n", id)
		return nil
	},
}

var taskHeadingCmd = &cobra.Command{
	Use:   "heading [id] [headingID|none]",
	Short: "Put a task under a heading, or take it out with 'none'",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("task ID", args[0])
		if err != nil {
			return err
		}
		headingID, err := parseOptionalID("heading ID", args[1])
		if err != nil {
			return err
		}
		if err := app.Tasks.SetHeading(id, headingID); err != nil {
			return err
		}
		fmt.Printf("Task %d updated.\n", id)
		return nil
	},
}

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
	taskAddCmd.Flags().Int64Var(&areaFlag, "area", 0, "Area ID (for a task outside any project)")
	taskAddCmd.Flags().Int64Var(&headingFlag, "heading", 0, "Heading ID (files the task in the heading's project)")
	taskAddCmd.Flags().Int64Var(&parentID, "parent", 0, "Parent task ID (creates a subtask)")
	taskAddCmd.Flags().StringVarP(&dueStr, "due", "d", "", "Due date (YYYY-MM-DD)")
	taskAddCmd.Flags().StringVarP(&priorityStr, "priority", "P", "medium", "Priority (low, medium, high, critical)")
	taskTagCmd.Flags().StringVarP(&tagColor, "color", "c", "", "Color for the tag")

	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskCancelCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskParentCmd)
	taskCmd.AddCommand(taskAreaCmd)
	taskCmd.AddCommand(taskHeadingCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskStopCmd)
	taskCmd.AddCommand(taskTagCmd)
	taskCmd.AddCommand(taskMoveCmd)
	taskCmd.AddCommand(taskHistoryCmd)
}
