// Command desktop is the Zenith desktop app: a thin Wails v3 shell (ADR 0001)
// around the shared core. It is the only package that imports Wails. The window
// hosts the React front end in desktop/frontend; every call from it lands on
// appapi.Service, which talks to internal/core exactly like the CLI and TUI do.
package main

import (
	"context"
	"embed"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v3/pkg/application"
	"github.com/yalcinumutt/zenith-cli/internal/appapi"
	"github.com/yalcinumutt/zenith-cli/internal/core"
	"github.com/yalcinumutt/zenith-cli/internal/notify"
	"github.com/yalcinumutt/zenith-cli/internal/storage"
)

// frontend/dist is built by `wails3 build` (or `pnpm build`) before compiling.
//
//go:embed all:frontend/dist
var assets embed.FS

// version is set at release time with -ldflags "-X main.version=...".
var version = "0.1.0-beta"

func init() {
	// A registered event gets a strongly typed TypeScript API from the binding generator.
	application.RegisterEvent[appapi.ChangeEvent]("zenith:change")
}

func main() {
	store, err := storage.NewSQLiteStore()
	if err != nil {
		log.Fatalf("zenith: could not open the database: %v", err)
	}
	defer store.Close()

	c := core.New(store, core.SystemClock{}, notify.System())

	home, _ := os.UserHomeDir()
	svc := appapi.New(c, core.SystemClock{}, filepath.Join(home, ".zenith", "desktop.json"), version)

	app := application.New(application.Options{
		Name:        "Zenith",
		Description: "Tasks, projects and time in one calm place",
		Services:    []application.Service{application.NewService(svc)},
		Assets:      application.AssetOptions{Handler: application.AssetFileServerFS(assets)},
		Mac:         application.MacOptions{ApplicationShouldTerminateAfterLastWindowClosed: true},
	})

	// Core events -> the front end, which invalidates its cached lists.
	unsubscribe := c.Subscribe(func(e core.Event) {
		if ev, ok := appapi.ToChangeEvent(e); ok {
			app.Event.Emit("zenith:change", ev)
		}
	})
	defer unsubscribe()

	// While the app is open it detects writes from the CLI or daemon, and fires
	// due reminders through the native Notifier (ADR 0011). Closing the window
	// quits the app, which stops both loops.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = c.WatchExternal(ctx, time.Second) }()
	go tick(ctx, c, app)

	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:            "Zenith",
		Width:            1120,
		Height:           760,
		MinWidth:         720,
		MinHeight:        480,
		BackgroundColour: application.NewRGB(250, 250, 249),
		URL:              "/",
	})

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}

// tick delivers due reminders once a second until ctx ends.
func tick(ctx context.Context, c *core.Core, app *application.App) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	warned := false
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-t.C:
			if _, err := c.Tick(now); err != nil && !warned {
				warned = true
				log.Printf("zenith: reminder delivery: %v", err)
			}
		}
	}
}
