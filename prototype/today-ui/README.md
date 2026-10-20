# PROTOTYPE — Zenith Today / Upcoming / sidebar UI (throwaway)

Answers wayfinder ticket "Prototype: Today, Upcoming and sidebar UI". Fake in-memory data, no persistence, no Go.
Do not promote this code; rewrite the winner properly against the generated `api/` client.

    pnpm install && pnpm prototype      # http://localhost:5199

Variants (`?variant=A|B|C`, or the bottom bar, or `[` / `]`):

- **A — Calm sidebar, inline expansion**: Things-3 feel. Roomy rows, one accent, tasks expand in place with a checklist.
- **B — Dense table + inspector**: 28px rows, metadata in columns, selection fills a right-hand inspector.
- **C — Top tabs, week board, quick look**: no sidebar; 7-day rail on Today, drag-a-card-to-reschedule board on Upcoming, Space opens a quick-look overlay.

Shared keys: ↑/↓ select · Space expand/quick look · → / ← open/close · x complete · ⌘/Ctrl+↑/↓ reorder · 1–6 jump to a list · Esc close.
Drag any row to reorder (fractional index, ADR 0007). Completing a task with open subtasks is refused; completing the last subtask completes the parent (GLOSSARY).
Views to try: `?view=today`, `upcoming`, `inbox`, `project:launch` (headings), `project:move`, `logbook`.
