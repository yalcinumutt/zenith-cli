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
var whenStr string
var eveningFlag bool
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

		if whenStr != "" {
			w, err := core.ParseWhen(whenStr, time.Now())
			if err != nil {
				return err
			}
			in.When = w
		}
		in.Deadline = dueStr
		in.Evening = eveningFlag

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

var taskWhenCmd = &cobra.Command{
	Use:   "when [id] [today|tomorrow|someday|anytime|YYYY-MM-DD]",
	Short: "Set when a task is meant to be worked on",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("task ID", args[0])
		if err != nil {
			return err
		}
		w, err := core.ParseWhen(args[1], time.Now())
		if err != nil {
			return err
		}
		if err := app.Tasks.SetWhen(id, w); err != nil {
			return err
		}
		if eveningFlag {
			if err := app.Tasks.SetEvening(id, true); err != nil {
				return err
			}
		}
		fmt.Printf("Task %d: %s\n", id, describeWhen(w, eveningFlag))
		return nil
	},
}

var taskDeadlineCmd = &cobra.Command{
	Use:   "deadline [id] [YYYY-MM-DD|none]",
	Short: "Set or clear a task's deadline",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("task ID", args[0])
		if err != nil {
			return err
		}
		date := args[1]
		if date == "none" {
			date = ""
		}
		if err := app.Tasks.SetDeadline(id, date); err != nil {
			return err
		}
		if date == "" {
			fmt.Printf("Task %d: deadline cleared\n", id)
		} else {
			fmt.Printf("Task %d: deadline %s\n", id, date)
		}
		return nil
	},
}

var (
	reorderAfter, reorderBefore int64
	reorderHeading              string
	reorderEvening, reorderDay  bool
)

var taskReorderCmd = &cobra.Command{
	Use:   "reorder [id]",
	Short: "Move a task between two neighbours, optionally to another heading or This Evening",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseID("task ID", args[0])
		if err != nil {
			return err
		}
		var p core.Placement
		if reorderAfter != 0 {
			p.AfterID = &reorderAfter
		}
		if reorderBefore != 0 {
			p.BeforeID = &reorderBefore
		}
		if reorderHeading != "" {
			hid, err := parseOptionalID("heading ID", reorderHeading)
			if err != nil {
				return err
			}
			p.Heading = &core.HeadingChange{HeadingID: hid}
		}
		if reorderEvening && reorderDay {
			return fmt.Errorf("use only one of --evening and --day")
		}
		if reorderEvening || reorderDay {
			v := reorderEvening
			p.Evening = &v
		}
		if err := app.Tasks.Place(id, p); err != nil {
			return err
		}
		fmt.Printf("Task %d moved.\n", id)
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

var taskUntagCmd = &cobra.Command{
	Use:   "untag [id] [tag_name]",
	Short: "Remove a tag from a task",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf(errMsgInvalidID, err)
		}
		if err := app.Tasks.Untag(id, args[1]); err != nil {
			return err
		}
		fmt.Printf("Removed tag '%s' from task %d\n", args[1], id)
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
	taskAddCmd.Flags().StringVarP(&dueStr, "deadline", "d", "", "Deadline (YYYY-MM-DD)")
	taskAddCmd.Flags().StringVar(&dueStr, "due", "", "Alias of --deadline")
	_ = taskAddCmd.Flags().MarkHidden("due")
	taskAddCmd.Flags().StringVarP(&whenStr, "when", "w", "", "When: today, tomorrow, someday, anytime or YYYY-MM-DD")
	taskAddCmd.Flags().BoolVar(&eveningFlag, "evening", false, "Put the task under This Evening (needs a scheduled --when)")
	taskAddCmd.Flags().StringVarP(&priorityStr, "priority", "P", "medium", "Priority (low, medium, high, critical)")
	taskWhenCmd.Flags().BoolVar(&eveningFlag, "evening", false, "Put the task under This Evening")
	taskReorderCmd.Flags().Int64Var(&reorderAfter, "after", 0, "Place the task right after this task ID")
	taskReorderCmd.Flags().Int64Var(&reorderBefore, "before", 0, "Place the task right before this task ID")
	taskReorderCmd.Flags().StringVar(&reorderHeading, "heading", "", "Move under this heading ID, or 'none'")
	taskReorderCmd.Flags().BoolVar(&reorderEvening, "evening", false, "Move to This Evening")
	taskReorderCmd.Flags().BoolVar(&reorderDay, "day", false, "Move back to the day section")
	taskTagCmd.Flags().StringVarP(&tagColor, "color", "c", "", "Color for the tag")

	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskCancelCmd)
	taskCmd.AddCommand(taskReopenCmd)
	taskCmd.AddCommand(taskParentCmd)
	taskCmd.AddCommand(taskAreaCmd)
	taskCmd.AddCommand(taskReorderCmd)
	taskCmd.AddCommand(taskWhenCmd)
	taskCmd.AddCommand(taskDeadlineCmd)
	taskCmd.AddCommand(taskHeadingCmd)
	taskCmd.AddCommand(taskDeleteCmd)
	taskCmd.AddCommand(taskStartCmd)
	taskCmd.AddCommand(taskStopCmd)
	taskCmd.AddCommand(taskTagCmd)
	taskCmd.AddCommand(taskUntagCmd)
	taskCmd.AddCommand(taskMoveCmd)
	taskCmd.AddCommand(taskHistoryCmd)
}
