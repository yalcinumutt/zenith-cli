# Contributing to Zenith CLI

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
- [Go](https://go.dev/dl/) 1.18+
- [SQLite3](https://sqlite.org/download.html)

### Building
```bash
go mod tidy
go build -o zenith main.go
```

### Testing
```bash
go test ./...
```

## 🎨 Styling Guidelines
- We use [Lipgloss](https://github.com/charmbracelet/lipgloss) for all terminal styling.
- Keep the UI consistent with existing themes in `internal/ui/styles.go`.

## 📜 Code of Conduct
Please be respectful and professional in all interactions. We follow the standard Contributor Covenant Code of Conduct.
