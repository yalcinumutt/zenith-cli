package notify

import (
	"errors"
	"strings"
	"testing"
)

func found(string) (string, error)   { return "/bin/tool", nil }
func missing(string) (string, error) { return "", errors.New("not found") }

func TestCommandPerPlatform(t *testing.T) {
	evil := `x" & do shell script "boom`
	cases := []struct {
		goos string
		want string
	}{{"darwin", "/bin/tool"}, {"linux", "/bin/tool"}, {"windows", "/bin/tool"}}
	for _, c := range cases {
		n := &Notifier{goos: c.goos, lookPath: found}
		cmd, err := n.command("Zenith", evil)
		if err != nil {
			t.Fatalf("%s: %v", c.goos, err)
		}
		// The message must travel as data, never inside script text.
		switch c.goos {
		case "darwin":
			if cmd.Args[len(cmd.Args)-2] != evil || strings.Contains(strings.Join(cmd.Args[:len(cmd.Args)-2], " "), "boom") {
				t.Errorf("darwin message not passed as argv: %v", cmd.Args)
			}
		case "linux":
			if cmd.Args[len(cmd.Args)-1] != evil {
				t.Errorf("linux args = %v", cmd.Args)
			}
		case "windows":
			if strings.Contains(cmd.Args[len(cmd.Args)-1], "boom") {
				t.Errorf("windows script contains message text")
			}
			if !strings.Contains(strings.Join(cmd.Env, "\n"), "ZENITH_MESSAGE="+evil) {
				t.Errorf("windows env missing message")
			}
		}
	}
}

func TestMissingCapabilityIsErrUnavailable(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows", "plan9"} {
		n := &Notifier{goos: goos, lookPath: missing}
		err := n.Notify("t", "m")
		if !errors.Is(err, ErrUnavailable) {
			t.Errorf("%s: err = %v, want ErrUnavailable", goos, err)
		}
	}
}
