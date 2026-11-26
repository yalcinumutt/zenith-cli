import { useDraggable } from "@dnd-kit/core";
import { CalendarPlusIcon, CheckIcon, GripVerticalIcon } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import type { Block, Task } from "../../api/client";
import { ESTIMATES, fmtDuration, scheduledMinutes } from "../../lib/plan";

interface Props {
  tasks: Task[];
  blocks: Block[];
  onEstimate: (task: Task, minutes: number) => void;
  onPlace: (task: Task) => void;
}

/** Today's tasks: set an effort estimate, then drag onto the timeline or press Place. */
export function PlanTasks({ tasks, blocks, onEstimate, onPlace }: Props) {
  if (tasks.length === 0) return <p className="py-6 text-sm text-muted-foreground">No open tasks for this day.</p>;
  return (
    <ul className="flex flex-col gap-1.5" aria-label="Tasks to plan">
      {tasks.map((t) => <PlanTask key={t.id} task={t} blocks={blocks} onEstimate={onEstimate} onPlace={onPlace} />)}
    </ul>
  );
}

function PlanTask({ task, blocks, onEstimate, onPlace }: { task: Task; blocks: Block[]; onEstimate: Props["onEstimate"]; onPlace: Props["onPlace"] }) {
  const { attributes, listeners, setNodeRef, isDragging } = useDraggable({ id: `task:${task.id}`, data: { taskId: task.id } });
  const placed = scheduledMinutes(blocks, task.id);
  const covered = task.estimate > 0 && placed >= task.estimate;
  const options = ESTIMATES.includes(task.estimate) || task.estimate === 0 ? ESTIMATES : [...ESTIMATES, task.estimate].sort((a, b) => a - b);

  return (
    <li ref={setNodeRef} className={cn("flex items-center gap-2 rounded-lg border bg-card px-2 py-2", isDragging && "opacity-40")}>
      <button
        type="button"
        aria-label={`Drag ${task.title} onto the timeline`}
        className="cursor-grab touch-none rounded p-0.5 text-muted-foreground hover:bg-muted active:cursor-grabbing"
        {...listeners}
        {...attributes}
      >
        <GripVerticalIcon className="size-4" />
      </button>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{task.title}</p>
        <p className="truncate text-xs text-muted-foreground">
          {task.projectName || "No project"}
          {placed > 0 && <> · {covered ? <CheckIcon className="inline size-3 text-primary" aria-label="fully scheduled" /> : null} {fmtDuration(placed)} placed</>}
        </p>
      </div>
      <select
        aria-label={`Estimate for ${task.title}`}
        value={task.estimate}
        onChange={(e) => onEstimate(task, Number(e.target.value))}
        className="h-7 rounded-md border bg-background px-1.5 text-xs text-muted-foreground outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
      >
        <option value={0}>Estimate</option>
        {options.map((m) => <option key={m} value={m}>{fmtDuration(m)}</option>)}
      </select>
      <Button variant="ghost" size="icon-xs" aria-label={`Place ${task.title} in the next free slot`} onClick={() => onPlace(task)}>
        <CalendarPlusIcon />
      </Button>
    </li>
  );
}
