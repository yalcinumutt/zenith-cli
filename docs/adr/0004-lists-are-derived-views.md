# The six lists are derived views, never stored state

Inbox, Today, Upcoming, Anytime, Someday and Logbook are computed from a Task's When (a sum type: anytime / today / scheduled(date) / someday), Deadline and Status — no `list` column exists anywhere. This replaces the two free-form date columns (`planned_date`, `due_date`) used today. A stored `list` column was rejected: it can disagree with the dates it was derived from, and under sync two devices can merge a task into a list its dates no longer justify (e.g. "Upcoming" with a date in the past). Deriving the lists in the core (see ADR 0002) also means the CLI, TUI and desktop app compute identical lists instead of each inventing its own filter.

Consequence: an incomplete Today task stays in Today rather than rolling back to Anytime when its date passes.
