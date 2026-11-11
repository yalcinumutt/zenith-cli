# Contributing to Zenith

First off, thank you for considering contributing to Zenith! It's people like you that make the open-source community such an amazing place to learn, inspire, and create.

## 🚩 How Can I Contribute?

### Reporting Bugs
- Use the **GitHub Issues** tab to report bugs.
- Include steps to reproduce the issue and your OS/terminal environment.

### Suggesting Enhancements
- Open a **GitHub Issue** with the "enhancement" label.
- Describe the feature and why it would be useful.

### Pull Requests
1. Fork the Project.
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`).
3. Commit your Changes (`git commit -m 'Add some AmazingFeature'`).
4. Push to the Branch (`git push origin feature/AmazingFeature`).
5. Open a Pull Request.

## 🛠️ Local Development Setup

### Prerequisites
- [Go](https://go.dev/dl/): the version in `go.mod`.
- Nothing else for the CLI: SQLite is the pure-Go `modernc.org/sqlite` driver, so **no C compiler (CGO) is needed**.
- For the desktop app only: [Node](https://nodejs.org/) 22+, [pnpm](https://pnpm.io/), and the [Wails v3](https://v3.wails.io/) CLI (`go install github.com/wailsapp/wails/v3/cmd/wails3@v3.1.0`). macOS and Linux builds of the desktop shell need CGO (Xcode command line tools / `gcc` plus GTK 4 and WebKitGTK 6.0 on Linux: `libgtk-4-dev libwebkitgtk-6.0-dev`); Windows does not.

### Building the CLI (no CGO)
```bash
CGO_ENABLED=0 go build -o zenith main.go
```

### Testing
```bash
go test $(go list ./... | grep -v '/desktop$')   # the CLI, TUI and shared core
```

### The desktop app
The app lives in `desktop/`: a thin Wails v3 shell (`desktop/main.go`, the only package that imports Wails), the Wails-free adapter `internal/appapi`, and a React + TypeScript front end in `desktop/frontend` (Vite, Tailwind, TanStack Query, Zustand, cmdk). The front end reaches Go only through `desktop/frontend/src/api/client.ts` (a lint rule forbids importing the generated bindings anywhere else).

```bash
cd desktop
wails3 dev          # run with hot reload
wails3 build        # production binary in desktop/bin
wails3 package      # macOS .app bundle, Windows installer, Linux AppImage/deb/rpm
```

To work on the UI against the real core in a normal browser (handy for screenshots and tests), run the shell in server mode and open http://localhost:8080:

```bash
cd desktop && wails3 task build:server && ./bin/Zenith-server   # or: go run -tags server .
```

Front-end checks (the same gate CI runs): `cd desktop/frontend && pnpm install && pnpm check` (typecheck, lint, 4.5:1 contrast check) and `pnpm build`. After changing `internal/appapi`, regenerate the TypeScript bindings with `cd desktop && wails3 generate bindings -clean=true -ts -i` (they are generated, not committed).

The accessibility baseline for the desktop app is in [docs/desktop-accessibility.md](docs/desktop-accessibility.md).

Desktop releases are cut by pushing a `desktop-v*` tag (`.github/workflows/desktop-release.yml`); CLI releases are cut by `v*` tags.

## 🎨 Styling Guidelines
- We use [Lipgloss](https://github.com/charmbracelet/lipgloss) for all terminal styling.
- Keep the UI consistent with existing themes in `internal/ui/styles.go`.

## 📜 Code of Conduct
Please be respectful and professional in all interactions. We follow the standard Contributor Covenant Code of Conduct.
