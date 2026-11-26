---
status: accepted
---

# Time Blocks are their own synced entity, not a time on the Task

Daily Planning places a Task at a time of day. `When` is a plain calendar date by design (ADR 0004), so the time cannot live there without breaking the derived lists and the "no time zone" rule. We add a `TimeBlock` entity (`task_id`, `date`, `start_min`, `duration_min`) with its own uuid, field stamps and tombstone, like every other synced row (ADR 0003, 0010), plus an `estimate_minutes` column on tasks.

## Considered options

- **Start time and estimate columns on Task:** simpler, but a task gets one slot per day, moving it rewrites the task's stamps, and a time of day leaks into the date-only When.
- **Store blocks as calendar events in an external calendar:** needs an account and breaks local-first use.

## Consequences

- Rules live in `core.PlanningService`: blocks never overlap on a day, never cross midnight, are at least 5 minutes, and only open tasks can be blocked. Placing a block does not change the task's When.
- A block of a deleted task reads as absent (the store joins on live tasks); a completed task keeps its blocks, shown as done.
- Calendar busy times (read-only ICS feed) and AI suggestions plug in later as inputs to the same `PlanningService`; neither writes blocks directly.
