import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { Flag, GripVertical, Repeat as RepeatIcon } from "lucide-react";
import type { Task } from "../api/client";
import { toggleComplete } from "../commands/actions";
import { deadlineLabel, friendlyDay } from "../lib/dates";
import { cn } from "../lib/utils";
import { useInfo } from "../state/queries";
import { useUI } from "../state/ui";
import { TaskCard, TitleEditor } from "./TaskCard";
import { RoundCheck } from "./ui";

interface Props {
  task: Task;
  section: string;
  /** Show the project name as a subtitle (lists that mix projects). */
  showProject?: boolean;
  /** Show the scheduled day as a chip. */
  showWhen?: boolean;
  /** Draw the insertion line above or below this row while a drag is over it. */
  insertion?: "above" | "below" | null;
}

export function TaskRow({ task, section, showProject, showWhen, insertion }: Props) {
  const today = useInfo().data?.today ?? "";
  const selected = useUI((s) => s.selectedId === task.id);
  const expanded = useUI((s) => s.expandedId === task.id);
  const { attributes, listeners, setNodeRef, transform, transition, isDragging, setActivatorNodeRef } = useSortable({ id: task.id, data: { section } });

  const done = task.status !== "open";
  const cancelled = task.status === "cancelled";

  return (
    <div
      ref={setNodeRef}
      data-task-id={task.id}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      onClick={() => useUI.getState().select(task.id)}
      onDoubleClick={() => useUI.getState().expand(expanded ? null : task.id)}
      className={cn(
        "group relative rounded-xl",
        isDragging && "opacity-40",
        expanded ? "my-2 bg-card shadow-[0_2px_14px_rgba(0,0,0,0.14)] ring-1 ring-line" : selected ? "bg-selected" : "hover:bg-hover",
        insertion === "above" && "before:absolute before:-top-px before:left-3 before:right-3 before:h-0.5 before:bg-accent",
        insertion === "below" && "after:absolute after:-bottom-px after:left-3 after:right-3 after:h-0.5 after:bg-accent",
      )}
    >
      <div className="flex items-start gap-3 px-3 py-2.5">
        <button
          ref={setActivatorNodeRef}
          {...attributes}
          {...listeners}
          aria-label={`Reorder ${task.title}`}
          className="absolute -left-5 top-3 hidden h-5 w-4 items-center justify-center rounded text-muted focus-visible:flex group-hover:flex"
        >
          <GripVertical className="h-3.5 w-3.5" aria-hidden />
        </button>
        <RoundCheck
          checked={done}
          label={`${task.status === "open" ? "Complete" : "Reopen"} ${task.title}`}
          onToggle={() => void toggleComplete(task)}
        />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline gap-x-2">
            {expanded ? (
              <div className="min-w-48 flex-1"><TitleEditor task={task} /></div>
            ) : (
              <span className={cn("text-[15px] leading-snug", done && "text-muted line-through", cancelled && "italic")}>{task.title}</span>
            )}
            {task.repeat && (
              <span className="inline-flex items-center" title={task.repeat.text}>
                <RepeatIcon className="h-3 w-3 translate-y-0.5 text-muted" aria-hidden />
                <span className="sr-only">Repeats: {task.repeat.text}</span>
              </span>
            )}
            {task.tags.map((t) => <span key={t.id} className="text-xs text-muted">#{t.name}</span>)}
            {!expanded && task.subtasksTotal > 0 && (
              <span className="text-xs text-muted" aria-label={`${task.subtasksTotal - task.subtasksOpen} of ${task.subtasksTotal} subtasks done`}>
                {task.subtasksTotal - task.subtasksOpen}/{task.subtasksTotal}
              </span>
            )}
          </div>
          {showProject && task.projectName && !expanded && <div className="text-xs text-muted">{task.projectName}</div>}
        </div>
        {task.deadline && task.status === "open" && (
          <span className={cn("flex shrink-0 items-center gap-1 pt-0.5 text-xs", task.overdue ? "font-medium text-danger" : "text-muted")}>
            <Flag className="h-3 w-3" aria-hidden />
            <span className="sr-only">Deadline </span>{deadlineLabel(task.deadline, today)}
          </span>
        )}
        {showWhen && task.whenKind === "scheduled" && task.whenDate && task.status === "open" && (
          <span className="shrink-0 pt-0.5 text-xs text-muted">{friendlyDay(task.whenDate, today)}</span>
        )}
      </div>
      {expanded && <TaskCard task={task} />}
    </div>
  );
}
