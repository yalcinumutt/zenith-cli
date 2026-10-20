---
status: accepted
---

# Build the desktop app on Wails v3 beta, with Wails confined to a thin shell

The desktop app needs a tray/menu-bar timer, a global quick-entry hotkey and autostart after v1. Wails v2 (stable) documents none of these; Wails v3 has them but is still beta (`v3.0.0-beta.28`, no GA date). We build on v3 and ship v1 publicly labelled beta (switching to GA if it lands first), because retrofitting these features onto v2 later would cost more than tracking a beta.

To keep the bet cheap, only a small shell package (window, tray, bindings glue) may import Wails. The shared core, storage and business logic stay plain Go, and the React app reaches Go through a single typed client module. If v3 breaks, only the shell is rewritten.

## Considered options

- **Wails v2 now, migrate later:** stable, but forces a migration and blocks the tray and hotkey features.
- **Wait for v3 GA before building:** safe, but stalls all UI and core work on an unknown date.

## Consequences

- Linux support is limited to recent mainstream distros (v3 defaults to GTK4/WebKitGTK 6 and drops the GTK3 path in v3.1); tray and global hotkey are best-effort there.
- Evidence and open questions: `docs/research/wails-cross-platform.md` on branch `research/wails-cross-platform`. Decision recorded in the Wails v2 or v3 beta ticket on the issue tracker.
