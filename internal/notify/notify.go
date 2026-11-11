// Package notify delivers desktop notifications on macOS, Windows and Linux by
// shelling out to each platform's native mechanism. It satisfies core.Notifier.
package notify

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

// ErrUnavailable means this machine cannot show native notifications (a tool
// is missing or the platform is unsupported). Callers degrade quietly: the
// reminder is still recorded and shown in the terminal.
var ErrUnavailable = errors.New("desktop notifications unavailable")

// Notifier shows notifications through the operating system.
type Notifier struct {
	goos     string
	lookPath func(string) (string, error)
}

// System returns a Notifier for the running OS.
func System() *Notifier {
	return &Notifier{goos: runtime.GOOS, lookPath: exec.LookPath}
}

// Notify shows a notification. It returns an error wrapping ErrUnavailable when
// the platform has no usable notification capability.
func (n *Notifier) Notify(title, message string) error {
	cmd, err := n.command(title, message)
	if err != nil {
		return err
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%w: %s failed: %v %s", ErrUnavailable, cmd.Args[0], err, trim(out))
	}
	return nil
}

func trim(b []byte) string {
	if len(b) > 200 {
		b = b[:200]
	}
	return string(b)
}

// command builds the platform command. Title and message reach the script as
// arguments or environment variables, never spliced into script text, so a
// reminder title cannot inject commands.
func (n *Notifier) command(title, message string) (*exec.Cmd, error) {
	switch n.goos {
	case "darwin":
		if _, err := n.lookPath("osascript"); err != nil {
			return nil, fmt.Errorf("%w: osascript not found", ErrUnavailable)
		}
		return exec.Command("osascript",
			"-e", "on run argv",
			"-e", `display notification (item 1 of argv) with title (item 2 of argv) sound name "Glass"`,
			"-e", "end run",
			message, title), nil
	case "linux":
		if _, err := n.lookPath("notify-send"); err != nil {
			return nil, fmt.Errorf("%w: install libnotify (notify-send) to enable them", ErrUnavailable)
		}
		return exec.Command("notify-send", "--app-name=Zenith", "--", title, message), nil
	case "windows":
		ps, err := n.lookPath("powershell.exe")
		if err != nil {
			return nil, fmt.Errorf("%w: powershell.exe not found", ErrUnavailable)
		}
		c := exec.Command(ps, "-NoProfile", "-NonInteractive", "-Command", windowsToast)
		c.Env = append(c.Environ(), "ZENITH_TITLE="+title, "ZENITH_MESSAGE="+message)
		return c, nil
	}
	return nil, fmt.Errorf("%w: unsupported platform %q", ErrUnavailable, n.goos)
}

// windowsToast raises a toast under PowerShell's own app id, which Windows
// accepts without registering a Start-menu shortcut for Zenith.
const windowsToast = `
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$xml = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$text = $xml.GetElementsByTagName('text')
$text.Item(0).AppendChild($xml.CreateTextNode($env:ZENITH_TITLE)) | Out-Null
$text.Item(1).AppendChild($xml.CreateTextNode($env:ZENITH_MESSAGE)) | Out-Null
$appId = '{1AC14E77-02E7-4E5D-B744-2EB1AE5198B7}\WindowsPowerShell\v1.0\powershell.exe'
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier($appId).Show([Windows.UI.Notifications.ToastNotification]::new($xml))
`
