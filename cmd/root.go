package cmd

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/notify"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

var (
	store   storage.Store
	app     *core.Core
	Version = "v0.1.2"
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
		app = core.New(s, core.SystemClock{}, notify.System())

		// Don't run passive check for the timer daemon itself to avoid loops
		if cmd.Name() != "daemon" {
			checkExpiredTimersPassive()
		}

		return nil
	},
}

// checkExpiredTimersPassive fires due reminders on any invocation through the
// same core path as the daemon.
func checkExpiredTimersPassive() {
	if app == nil {
		return
	}
	fired, err := app.Tick(time.Now())
	reportNotifyError(err)
	if len(fired) == 0 {
		return
	}

	fmt.Print("\a") // Terminal bell sound
	fmt.Println()
	for _, r := range fired {
		alertStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#FF0000")).
			Padding(0, 2).
			MarginBottom(1)
		fmt.Println(alertStyle.Render(fmt.Sprintf("🔔 REMINDER: %s", r.Title)))
	}
}

var warnedUnavailable bool

// reportNotifyError explains a notification failure once per process. A
// missing capability is expected on some machines, so it never aborts anything.
func reportNotifyError(err error) {
	if err == nil || warnedUnavailable {
		return
	}
	warnedUnavailable = true
	if errors.Is(err, notify.ErrUnavailable) {
		fmt.Fprintf(os.Stderr, "note: %v (reminders still show in the terminal)\n", err)
		return
	}
	fmt.Fprintf(os.Stderr, "note: could not deliver a notification: %v\n", err)
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
