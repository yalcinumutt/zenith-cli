package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/yalcinumut/zenith-cli/internal/storage"
)

var (
	store   storage.Store
	Version = "v0.1.0"
)

var rootCmd = &cobra.Command{
	Use:     "zenith",
	Version: Version,
	Short:   "Zenith is a CLI Productivity Tool",
	Long: `Zenith is a powerful CLI for habit tracking, task management, and project management.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		s, err := storage.NewSQLiteStore()
		if err != nil {
			return err
		}
		store = s

		// Don't run passive check for the timer daemon itself to avoid loops
		if cmd.Name() != "daemon" {
			checkExpiredTimersPassive()
		}

		return nil
	},
}

func checkExpiredTimersPassive() {
	if store == nil {
		return
	}
	timers, err := store.GetPendingScheduledTimers()
	if err != nil {
		return
	}
	now := time.Now()
	var triggered []string
	for _, t := range timers {
		if t.TriggerTime.Before(now) {
			triggered = append(triggered, t.Title)
			_ = store.MarkScheduledTimerTriggered(t.ID)
			// Trigger macOS notification
			script := fmt.Sprintf(`display notification "%s" with title "Zenith Reminder" sound name "Glass"`, t.Title)
			_ = exec.Command("osascript", "-e", script).Run()
		}
	}

	if len(triggered) > 0 {
		fmt.Print("\a") // Terminal bell sound
		fmt.Println()
		for _, title := range triggered {
			alertStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FF0000")).
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FF0000")).
				Padding(0, 2).
				MarginBottom(1)
			fmt.Println(alertStyle.Render(fmt.Sprintf("🔔 REMINDER: %s", title)))
		}
	}
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Root flags if any
}
