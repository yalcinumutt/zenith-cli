# Subtasks reuse the full Task model, with no depth limit

A Subtask is a Task whose container is `parent_task_id` rather than a Project/Heading/none, carrying its own When, Deadline, Project, Tags and Timer rather than a stripped-down checklist-item entity. Nesting is unlimited. This was a deliberate deviation from the simpler alternative (a dedicated, flat Checklist-item entity with just a title and done flag): the richer model was chosen so a subtask can carry real dates and its own filing, at the cost of every list query needing to exclude subtasks from top-level and cross-project views (a subtask is visible only nested under its parent, and in its own Project's own list).

Two consequences that had to be decided alongside this:

- **Completion is single-writer.** Completing the last open Subtask runs the same core completion path a manual completion does, so both triggers converge on one write rather than racing; a Task with open Subtasks cannot be completed directly (blocked by the core, not raced under sync).
- **The parent chain must stay acyclic.** Unlimited nesting means the core rejects any `parent_task_id` write that would make a task its own ancestor — required for both list rendering and the completion walk above to terminate.
