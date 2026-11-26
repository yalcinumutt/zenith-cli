---
status: accepted
---

# Planner inputs are injected, and every suggestion is validated and reviewed

Daily Planning gains two optional inputs: busy times from a read-only ICS calendar feed, and block suggestions from an AI model. `core.PlanningService` depends only on two small interfaces, `BusySource` and `Advisor`; the HTTP, iCalendar and model code lives in `internal/calendar` and `internal/ai` and is wired in by `appapi` (ADR 0002).

## Decisions

- **ICS feed, not OAuth.** One secret iCal URL works for Google, Outlook and iCloud on every OS and needs no registered app. It is read-only; Zenith never writes to a calendar. A failed refresh keeps the last good copy and surfaces an error instead of emptying the day.
- **Suggestions are never written directly.** `Suggest` returns proposals; `Apply` places them all-or-nothing in one transaction and re-checks overlaps, so a day that changed in between is refused rather than double-booked.
- **The core distrusts the Advisor.** Proposals for unknown tasks, outside working hours, in the past, on busy time, overlapping each other, off the 5-minute grid or over a task's remaining minutes are dropped and reported. A deterministic scheduler (urgency, priority, earliest free fit) works with no AI at all.
- **What leaves the machine.** Only task titles, remaining minutes, priority, deadlines and anonymous busy spans go to a model. Notes, projects, tags and calendar event titles never do. Cloud providers are opt-in and the UI says so; a local model sends nothing off the computer.
- **Secrets.** API keys live in the system keychain and are write-only from the UI. The calendar link, which is also a secret, is kept in `planning.json` with owner-only permissions.

## Consequences

- Event times without a zone are read in the clock's zone; recurring events are expanded in their own zone so wall-clock times survive daylight saving. RRULE support covers DAILY, WEEKLY (BYDAY), MONTHLY (BYMONTHDAY) and YEARLY with INTERVAL, COUNT, UNTIL, EXDATE and RECURRENCE-ID overrides; other rules are ignored.
- Calendar events are advisory on manual placement: the user may block time over an event; only suggestions avoid them.
