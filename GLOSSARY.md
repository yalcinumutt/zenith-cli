# Zenith

A personal productivity tool for tasks, projects, habits and time tracking, used through a CLI, a terminal dashboard and a desktop app over one shared body of data.

## Language

**Core**:
The single place where Zenith's rules live (what completing a task, starting a timer or firing a reminder means). Every front-end goes through it.
_Avoid_: Backend, engine, logic layer

**Front-end**:
A way of using Zenith: the CLI, the terminal dashboard (TUI) or the desktop app. Front-ends present and collect; they never decide.
_Avoid_: Client, UI, interface

**Reminder**:
A moment the user asked to be told about. It becomes due once, and is delivered by whichever front-end notices first.
_Avoid_: Scheduled timer, alarm, alert

**Area**:
A long-lived bucket of responsibility ("Work", "Health") that holds Projects and top-level Tasks. Areas never complete.
_Avoid_: Category, folder

**Project**:
A finite outcome with its own completion state, living inside an Area (or standalone). Holds Headings and Tasks.

**Heading**:
A named section inside one Project that groups Tasks for display. A Heading cannot hold anything but Tasks, and cannot itself be completed.
_Avoid_: Section, group

**Task**:
A single actionable thing with a When and an optional Deadline. May have Subtasks nested under it to any depth, Tags, and a running Timer.

**Subtask**:
A Task whose container is another Task (its parent) rather than a Project, Heading, or nothing. Carries the full Task model — its own When, Deadline, Project, Tags — and is hidden from every list except nested display under its parent and its own Project's list. Completing the last open Subtask of a Task completes that Task automatically, through the same completion rule a manual completion uses; a Task with open Subtasks cannot be completed directly.
_Avoid_: Checklist item, sub-item

**When**:
A Task's intended working time: Anytime (unset), Today, a specific future date, or Someday. Distinct from Deadline. The six lists (Inbox, Today, Upcoming, Anytime, Someday, Logbook) are computed from When, Deadline and Status — none of them is ever stored on the row.
_Avoid_: Planned date, scheduled date

**Deadline**:
The date a Task is actually due, independent of When.
_Avoid_: Due date

**Status**:
A Task or Project's completion state: open, completed, or cancelled. "In progress" is not a Status — it's derived from whether a Timer is currently running against the Task.
_Avoid_: In progress (as a stored state)

**Repeater**:
The rule and template behind a recurring Task. Each occurrence it produces is an ordinary Task row; completing one is a normal completion, and the Repeater generates the next.
_Avoid_: Recurring task (for the rule itself — that term is for the occurrences it produces)
