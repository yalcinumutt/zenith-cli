# Zenith CLI

**Zenith** is a powerful, open-source productivity tool for the terminal. It combines **Task Management**, **Habit Tracking**, and **Project Management** into a single, high-performance Go application with a stunning Terminal User Interface (TUI).

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
- **🗄️ SQLite Storage**: Fast, reliable, and relational data management.

## 🚀 Installation

### From Source

Ensure you have [Go](https://go.dev/) installed (v1.18+).

```bash
git clone https://github.com/yalcinumut/zenith-cli.git
cd zenith-cli
go build -o zenith main.go
mv zenith /usr/local/bin/ # Or any directory in your PATH
```

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
```

### Habits

```bash
zenith habit add "Drink 2L Water"
zenith habit log 1               # Log completion for today
zenith habit list                # View streaks and progress
```

### Projects

```bash
zenith project add "Open Source Zenith"
zenith project list
```

### Dashboard & Utilities

```bash
zenith tui      # Launch interactive mode (press 's' to toggle timers)
zenith plan     # Interactive daily planning session
zenith log      # View historical activity timeline
zenith summary  # Get your daily brief grouped by projects
zenith search "go" # Search across tasks and habits
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
