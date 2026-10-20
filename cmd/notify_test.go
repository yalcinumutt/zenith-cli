package cmd

import "testing"

func TestNotificationCommand(t *testing.T) {
	if c := notificationCommand("darwin", "Stand up"); c == nil || c.Args[0] != "osascript" {
		t.Errorf("darwin should use osascript, got %v", c)
	}
	for _, goos := range []string{"windows", "linux"} {
		if c := notificationCommand(goos, "Stand up"); c != nil {
			t.Errorf("%s has no native notifier yet and must be a quiet no-op, got %v", goos, c.Args)
		}
	}
}
