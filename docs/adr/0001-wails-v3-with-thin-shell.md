---
status: accepted
---

# Build the desktop app on Wails v3, with Wails confined to a thin shell

The desktop app needs a tray/menu-bar timer, a global quick-entry hotkey and autostart after v1. Wails v2 (stable) documents none of these; Wails v3 has them and is now stable (`v3.0.0` GA on 2026-10-26, `v3.1.0` on 2026-11-01). This ADR was first written when v3 was still beta (`v3.0.0-beta.28`); GA removed that risk. We build on the latest v3 release, because retrofitting these features onto v2 later would cost more than starting on v3.

To keep a framework change cheap, only a small shell package (window, tray, bindings glue) may import Wails. The shared core, storage and business logic stay plain Go, and the React app reaches Go through a single typed client module. If v3 changes incompatibly, only the shell is rewritten.

## Considered options

- **Wails v2 now, migrate later:** stable, but forces a migration and blocks the tray and hotkey features.
- **Wait for v3 GA before building:** was the safe option while v3 was beta; moot now that GA has shipped.

## Consequences

- Linux support is limited to recent mainstream distros (v3 defaults to GTK4/WebKitGTK 6 and drops the GTK3 path as of v3.1); tray and global hotkey are best-effort there.
- Evidence and open questions: `docs/research/wails-cross-platform.md` on branch `research/wails-cross-platform`. Decision recorded in the Wails v2 or v3 beta ticket on the issue tracker.
