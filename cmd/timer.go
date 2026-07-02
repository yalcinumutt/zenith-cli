package cmd

import (
	"fmt"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/models"
	"github.com/yalcinumut/zenith-cli/internal/ui"
)

var timerCmd = &cobra.Command{
	Use:   "timer",
	Short: "Manage and schedule task timers and reminders",
}

var taskIDFlag int64

var timerAddCmd = &cobra.Command{
	Use:   "add [duration] [message]",
	Short: "Add a scheduled reminder/timer (e.g. 25m, 1h, 10s)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		durationStr := args[0]
		message := "Time is up!"
		if len(args) > 1 {
			message = args[1]
		}

		duration, err := time.ParseDuration(durationStr)
		if err != nil {
			return fmt.Errorf("invalid duration format (e.g. 25m, 10s, 1h): %w", err)
		}

		triggerTime := time.Now().Add(duration)
		timer := &models.ScheduledTimer{
			Title:       message,
			TriggerTime: triggerTime,
			Status:      "pending",
		}

		if taskIDFlag != 0 {
			timer.TaskID = &taskIDFlag
		}

		if err := store.AddScheduledTimer(timer); err != nil {
			return err
		}

		fmt.Printf("Timer scheduled! ID: %d. Will trigger in %s (at %s).\n", 
			timer.ID, duration, triggerTime.Format("15:04:05"))
		return nil
	},
}

var timerListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all pending scheduled timers",
	RunE: func(cmd *cobra.Command, args []string) error {
		timers, err := store.GetPendingScheduledTimers()
		if err != nil {
			return err
		}

		if len(timers) == 0 {
			fmt.Println("No pending timers scheduled.")
			return nil
		}

		fmt.Println(ui.HeaderStyle.Render(" Scheduled Timers "))
		fmt.Printf("%-3s | %-20s | %-19s | %s\n", "ID", "Message", "Trigger Time", "Time Remaining")
		fmt.Println("----|----------------------|---------------------|----------------")
		for _, t := range timers {
			remaining := time.Until(t.TriggerTime).Round(time.Second)
			var remStr string
			if remaining < 0 {
				remStr = "Expired"
			} else {
				remStr = remaining.String()
			}

			fmt.Printf("%-3d | %-20s | %-19s | %s\n", 
				t.ID, t.Title, t.TriggerTime.Format("2006-01-02 15:04:05"), remStr)
		}
		return nil
	},
}

var timerDaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run active timer daemon checking for expired timers in the background",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(ui.HeaderStyle.Render(" Zenith Timer Daemon "))
		fmt.Println("Checking for expired timers... Press Ctrl+C to stop.")

		for {
			timers, err := store.GetPendingScheduledTimers()
			if err == nil {
				now := time.Now()
				for _, t := range timers {
					if t.TriggerTime.Before(now) {
						// Trigger notification
						triggerDesktopNotification(t.Title)

						// Mark as triggered
						_ = store.MarkScheduledTimerTriggered(t.ID)

						// Log to stdout
						fmt.Printf("[%s] 🔔 TIMER TRIGGERED: %s\n", 
							now.Format("15:04:05"), ui.TitleStyle.Render(t.Title))
					}
				}
			}
			time.Sleep(1 * time.Second)
		}
	},
}

func triggerDesktopNotification(message string) {
	// macOS native notification using AppleScript
	script := fmt.Sprintf(`display notification "%s" with title "Zenith Reminder" sound name "Glass"`, message)
	cmd := exec.Command("osascript", "-e", script)
	_ = cmd.Run()

	// Terminal bell/alert sound
	fmt.Print("\a")
}

func init() {
	timerAddCmd.Flags().Int64VarP(&taskIDFlag, "task", "t", 0, "Associate with task ID")

	rootCmd.AddCommand(timerCmd)
	timerCmd.AddCommand(timerAddCmd)
	timerCmd.AddCommand(timerListCmd)
	timerCmd.AddCommand(timerDaemonCmd)
}
