# Desktop app: accessibility and theming baseline

The desktop app (`desktop/`) meets this baseline. Treat it as a checklist for every UI change.

## Checklist

- [ ] **Everything is reachable by keyboard.** Every action is a command in `desktop/frontend/src/commands/registry.ts`, available from the Mod+K palette and, where it has one, a shortcut. Mod is Command on macOS and Control elsewhere. Tab order follows the visual order; dialogs trap focus and return it on close (Radix).
- [ ] **A visible focus ring** on every interactive element (`:focus-visible`, 2px accent outline, in `src/index.css`). Never remove it.
- [ ] **Semantics come from Radix or native elements.** Task completion is a Radix `checkbox` (role `checkbox`), dialogs are Radix `dialog`, the sidebar is a `nav` with `aria-current="page"` on the open list, headings are `h1`/`h2`, selects/inputs are native.
- [ ] **Accessible names.** Every task checkbox is labelled `Complete <title>` / `Reopen <title>`; icon-only buttons have an `aria-label` (reorder grips, rename/delete, clear deadline, remove tag); decorative icons are `aria-hidden`; the toast is an `aria-live="polite"` status region.
- [ ] **Text contrast is at least 4.5:1 in both themes.** Colours are defined once as CSS variables in `src/index.css`; `pnpm contrast` (part of `pnpm check` and CI) computes the WCAG ratio for every text/background pair and fails below 4.5:1.
- [ ] **Reduced motion is respected.** A `prefers-reduced-motion: reduce` rule disables all animation and transitions.
- [ ] **Colour is never the only signal.** Overdue deadlines show the text "N d overdue" with a flag icon as well as red; completion shows a check mark and strikethrough.
- [ ] **Red or orange only for overdue Deadlines** (and destructive buttons). One accent colour; a muted palette otherwise; the system font stack.

## Themes

Light, dark and system. **System** follows `prefers-color-scheme` and is the default. The manual override is stored in the app's own preferences file (`~/.zenith/desktop.json`, written by `internal/appapi`), **not** in browser storage, so it survives relaunches and web-view data resets. Change it from the sidebar's theme button or the palette ("Change theme").

## How to check

```bash
cd desktop/frontend
pnpm check          # typecheck + lint + contrast
```

Manual pass for larger changes: tab through the whole window without a mouse, run a task through complete/expand/edit/reorder by keyboard, switch themes, and enable "reduce motion" in the OS.
