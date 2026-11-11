## Zenith Desktop (beta)

A free, open-source desktop app for tasks, projects and time, built on the same core as the `zenith` CLI and terminal dashboard. Your data lives in `~/.zenith/data.db`; the CLI, TUI and desktop app all read and write it.

### Install

| OS | Download | Notes |
| --- | --- | --- |
| macOS (Apple silicon) | `Zenith-<version>-macos-arm64.dmg` | Open the dmg and drag Zenith to Applications. |
| macOS (Intel) | `Zenith-<version>-macos-amd64.dmg` | Same. |
| Windows 10/11 (64-bit) | `Zenith-<version>-windows-amd64.exe` | Run the installer. Windows may show SmartScreen: choose "More info" then "Run anyway" (the beta is not code-signed). |
| Linux (64-bit) | `Zenith-<version>-linux-amd64.AppImage`, `.deb` or `.rpm` | AppImage: `chmod +x` and run. Needs a recent distribution (GTK 4 and WebKitGTK 6.0). |

**macOS builds may be unsigned.** Until signing is set up, macOS will say the app "can't be opened". Right-click (or Control-click) Zenith in Applications, choose **Open**, then **Open** again. You only need to do this once.

### Known limitations

- **This is a beta.** Expect rough edges; please report them at https://github.com/yalcinumutt/zenith-cli/issues.
- **Reminders only fire while the Zenith app or `zenith daemon` is running.** Closing the window quits the app; there is no tray or background mode yet.
- **Linux support is for recent distributions** (GTK 4 / WebKitGTK 6.0). Tray icons and global shortcuts are not part of v1.
- Windows and Linux builds are not code-signed.
- Sync between devices is not part of v1; it is the first planned feature after it.

### Support the project

Zenith is free and MIT licensed. If it is useful to you, consider [sponsoring on GitHub](https://github.com/sponsors/yalcinumutt).
