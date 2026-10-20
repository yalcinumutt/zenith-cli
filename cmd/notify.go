package cmd

import (
	"fmt"
	"os/exec"
	"runtime"
)

// notificationCommand returns the native notifier for goos, or nil when the
// platform has none yet (a quiet no-op until the Notifier ticket lands).
func notificationCommand(goos, message string) *exec.Cmd {
	if goos != "darwin" {
		return nil
	}
	script := fmt.Sprintf(`display notification "%s" with title "Zenith Reminder" sound name "Glass"`, message)
	return exec.Command("osascript", "-e", script)
}

func sendDesktopNotification(message string) {
	if c := notificationCommand(runtime.GOOS, message); c != nil {
		_ = c.Run()
	}
}
