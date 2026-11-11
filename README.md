# Zenith

**Zenith** is a powerful, open-source productivity tool with a **desktop app (beta)**, a terminal dashboard and a CLI, all on one shared SQLite database. It combines **Task Management**, **Habit Tracking**, and **Project Management** into a single, high-performance Go application with a stunning Terminal User Interface (TUI).

Inspired by `taskbook`, Zenith takes your productivity to the next level with SQLite storage, interactive dashboards, and advanced features like recurring tasks and daily summaries.

![Zenith Banner](https://via.placeholder.com/800x200?text=Zenith+CLI+-+Productivity+At+Your+Fingertips)

## ✨ Features

- **✅ Task Management**: Add, complete, prioritize, and delete tasks.
- **⏱️ Time Tracking**: Track time spent on tasks with start/stop timers.
- **🏷️ Tagging System**: Organize tasks with custom-colored tags.
- **📅 Daily Planning**: Plan your day interactively with `zenith plan`.
- **🔄 Recurring Tasks**: Set tasks to repeat daily, weekly, or monthly.
- **📊 Habit Tracking**: Track your daily streaks and maintain consistency.
- **🏗️ Project Boards**: Group tasks into projects/boards for better organization.
- **📜 Activity Log**: View a historical timeline of finished tasks.
- **📊 Daily Summary**: Get a snapshot of your day with `zenith summary`.
- **💻 Interactive TUI**: A full-screen dashboard for focused work (`zenith tui`).
- **🗄️ SQLite Storage**: Fast, reliable, and relational data management (pure Go, no CGO).
- **🖥️ Desktop app (beta)**: a calm, keyboard-first Things-style app on the same data (see below).
- **📂 Areas, Projects, Headings and Subtasks**, When vs Deadline, the six lists (Inbox, Today, Upcoming, Anytime, Someday, Logbook) and repeating tasks.

## 🖥️ Desktop app (beta)

The desktop app is a free beta. It shares `~/.zenith/data.db` with the CLI and the TUI, so a task added in one shows up in the others straight away.

Download it from the [Releases](https://github.com/yalcinumutt/zenith-cli/releases) page (releases tagged `desktop-v*`):

| OS | File | Install |
| --- | --- | --- |
| macOS | `Zenith-<version>-macos-arm64.dmg` / `-macos-amd64.dmg` | Open the dmg and drag Zenith to Applications. **Builds may be unsigned**: right-click Zenith, choose **Open**, then **Open** again (once). |
| Windows 10/11 | `Zenith-<version>-windows-amd64.exe` | Run the installer. SmartScreen may warn because the beta is not signed: "More info" → "Run anyway". |
| Linux | `Zenith-<version>-linux-amd64.AppImage`, `.deb`, `.rpm` | AppImage: `chmod +x` and run. Needs a recent distribution (GTK 4 and WebKitGTK 6.0). |

**Known limitations**

- It is a beta; expect rough edges.
- **Reminders only fire while the Zenith app or `zenith daemon` is running.** Closing the window quits the app. Run `zenith timer daemon` in a terminal to keep reminders firing in the background.
- Linux support is for recent distributions; tray icons and global shortcuts are not in v1.
- Device-to-device sync is not in v1 (it is the first planned feature after it).

Keyboard-first: `⌘/Ctrl+K` opens the command palette, `⌘/Ctrl+N` adds a to-do, `⌘/Ctrl+1…6` jump between the six lists, `Space` expands a task, `x` completes it and `⌘/Ctrl+↑/↓` reorders it.

If you find it useful, please consider [sponsoring the project](https://github.com/sponsors/yalcinumutt).

## 🚀 Installation

### From Source

Ensure you have [Go](https://go.dev/) installed (the version in `go.mod`). The CLI is pure Go and needs **no C compiler**.

```bash
git clone https://github.com/yalcinumutt/zenith-cli.git
cd zenith-cli
go build -o zenith main.go
mv zenith /usr/local/bin/ # Or any directory in your PATH
```

### Pre-built Binaries

You can download pre-built binaries for Windows, Linux, and macOS from the [Releases](https://github.com/yalcinumutt/zenith-cli/releases) page.

#### Linux (Debian/Ubuntu)

```bash
sudo dpkg -i zenith-cli_linux_amd64.deb
```

#### macOS (Homebrew)

*Coming soon! In the meantime, download the Darwin binary from releases.*

#### Windows

Download the `.zip` from releases, extract `zenith.exe`, and add it to your PATH.

## 🛠️ Usage

### Tasks

```bash
# Add tasks with project, due date, and priority

zenith task add "Finish the Go project" -p 1 -d "2026-03-20" --priority "high"

# Time tracking
zenith task start 1  # Start timer for task #1
zenith task stop 1   # Stop timer and record duration

# Tagging

zenith task tag 1 "urgent" --color "#FF5733"

zenith task list     # View all tasks (includes RUNNING status and timers)
zenith task done 1   # Complete task #1

# Move tasks between projects
zenith task move 1 2 # Move task #1 to project #2
```

### Habits

```bash
zenith habit add "Drink 2L Water"
zenith habit log 1               # Log completion for today
zenith habit list                # View streaks and progress
```

### Projects

```bash
zenith area add "Work"
zenith project add "Open Source Zenith" --area 1
zenith project list
zenith heading add 1 "Phase 1"          # sections inside a project
zenith task add "Write docs" --heading 1 --when today
zenith task add "Review" --parent 5      # a subtask
zenith task repeat 5 --weeks 1 --on mon,thu
zenith list today                        # inbox | today | upcoming | anytime | someday | logbook
```

### Dashboard & Utilities

```bash
zenith tui      # Launch interactive mode (press 's' to toggle timers)
zenith plan     # Interactive daily planning session
zenith log      # View historical activity timeline
zenith summary  # Get your daily brief grouped by projects
zenith search "go" # Search across tasks and habits
zenith update    # Check for and install the latest version
```

## 🎨 Technology Stack

- **Go**: Core logic and performance.
- **Cobra**: Robust CLI command routing.
- **Bubble Tea**: Interactive TUI components.
- **Lipgloss**: Terminal styling and layouts.
- **SQLite**: Local relational database.

## 🤝 Contributing

We love contributions! Please see our [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to get started.

## 📄 License

Distributed under the MIT License. See `LICENSE` for more information.
