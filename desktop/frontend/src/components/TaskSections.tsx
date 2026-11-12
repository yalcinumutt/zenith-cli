import {
  DndContext, KeyboardSensor, PointerSensor, closestCenter, useDroppable, useSensor, useSensors,
  type CollisionDetection, type DragEndEvent, type DragOverEvent, type DragStartEvent,
} from "@dnd-kit/core";
import { SortableContext, sortableKeyboardCoordinates, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { GripVertical } from "lucide-react";
import { Button } from "@/components/ui/button";
import { pointerOnly } from "@/lib/utils";
import { useEffect, useMemo, useState, type ReactNode } from "react";
import { api, type Task } from "../api/client";
import { attempt } from "../commands/actions";
import { useUI, type Row } from "../state/ui";
import { TaskRow } from "./TaskRow";

export interface Section {
  key: string;
  title?: ReactNode;
  /** Heading actions shown beside the title. */
  actions?: ReactNode;
  tasks: Task[];
  /** Dropping here files the task under this heading (null = no heading). Omit if sections aren't headings. */
  heading?: number | null;
  /** Dropping here sets the This Evening flag. Omit if sections aren't day/evening. */
  evening?: boolean;
  /** Text for an empty section; if omitted an empty section is not drawn. */
  empty?: string;
}

const headingId = (id: number) => `heading:${id}`;

// A heading drag only collides with other headings; a task drag ignores them.
const collide: CollisionDetection = (args) => {
  const heading = args.active.data.current?.type === "heading";
  return closestCenter({ ...args, droppableContainers: args.droppableContainers.filter((c) => (c.data.current?.type === "heading") === heading) });
};

interface Props {
  sections: Section[];
  showProject?: boolean;
  showWhen?: boolean;
  /** Quiet empty state when there are no tasks anywhere. */
  emptyText: string;
}

/**
 * The shared task list: titled sections of sortable rows. Dragging (or the
 * grip's keyboard handle) moves a task between neighbours, across headings and
 * between the day and evening sections, as one `PlaceTask` call (a single-row write).
 */
export function TaskSections({ sections, showProject, showWhen, emptyText }: Props) {
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const [overId, setOverId] = useState<number | string | null>(null);
  const [activeId, setActiveId] = useState<number | null>(null);
  const [activeHeading, setActiveHeading] = useState<string | null>(null);

  // Publish what is on screen so keyboard navigation and Mod+Up/Down know the order.
  const rows: Row[] = useMemo(() => sections.flatMap((s) => s.tasks.map((task) => ({ task, section: s.key }))), [sections]);
  useEffect(() => { useUI.getState().setRows(rows); return () => useUI.getState().setRows([]); }, [rows]);

  const sectionOf = (taskId: number) => sections.find((s) => s.tasks.some((t) => t.id === taskId));

  const headed = sections.filter((s) => typeof s.heading === "number");

  /** Reorder a heading among its siblings. */
  function endHeadingDrag(e: DragEndEvent) {
    if (!e.over || e.over.id === e.active.id) return;
    const id = Number(String(e.active.id).slice("heading:".length));
    const over = Number(String(e.over.id).slice("heading:".length));
    const ids = headed.map((s) => s.heading as number);
    const ordered = ids.filter((n) => n !== id);
    const overIndex = ordered.indexOf(over);
    const slot = ids.indexOf(id) < ids.indexOf(over) ? overIndex + 1 : overIndex;
    void attempt(() => api.reorderHeading(id, ordered[slot - 1] ?? null, ordered[slot] ?? null));
  }

  function handleEnd(e: DragEndEvent) {
    setOverId(null);
    setActiveId(null);
    setActiveHeading(null);
    if (e.active.data.current?.type === "heading") { endHeadingDrag(e); return; }
    if (!e.over) return;
    const id = Number(e.active.id);
    const source = sectionOf(id);
    if (!source) return;

    let target: Section | undefined;
    let ordered: Task[];
    let after: number | null;
    let before: number | null = null;
    const overRaw = e.over.id;

    if (typeof overRaw === "string" && overRaw.startsWith("section:")) {
      target = sections.find((s) => s.key === overRaw.slice("section:".length));
      if (!target) return;
      ordered = target.tasks.filter((t) => t.id !== id);
      after = ordered[ordered.length - 1]?.id ?? null; // dropped on a section: goes to its end
    } else {
      const overId = Number(overRaw);
      if (overId === id) return;
      target = sectionOf(overId);
      if (!target) return;
      ordered = target.tasks.filter((t) => t.id !== id);
      const overIndex = ordered.findIndex((t) => t.id === overId);
      const movingDown = target === source && source.tasks.findIndex((t) => t.id === id) < source.tasks.findIndex((t) => t.id === overId);
      const slot = movingDown ? overIndex + 1 : overIndex; // index the task will occupy
      after = ordered[slot - 1]?.id ?? null;
      before = ordered[slot]?.id ?? null;
    }

    const change: Parameters<typeof api.placeTask>[1] = { afterId: after, beforeId: before };
    if (target.key !== source.key) {
      if (target.heading !== undefined) { change.setHeading = true; change.headingId = target.heading; }
      if (target.evening !== undefined) { change.setEvening = true; change.evening = target.evening; }
    }
    void attempt(() => api.placeTask(id, change));
  }

  const handleOver = (e: DragOverEvent) => setOverId(e.over ? e.over.id : null);
  const handleStart = (e: DragStartEvent) => {
    if (e.active.data.current?.type === "heading") setActiveHeading(String(e.active.id));
    else setActiveId(Number(e.active.id));
  };

  const total = sections.reduce((n, s) => n + s.tasks.length, 0);
  const visible = sections.filter((s) => s.tasks.length > 0 || s.empty || s.title);

  return (
    <DndContext sensors={sensors} collisionDetection={collide} onDragStart={handleStart} onDragOver={handleOver} onDragEnd={handleEnd} onDragCancel={() => { setOverId(null); setActiveId(null); setActiveHeading(null); }}>
      {total === 0 && sections.every((s) => !s.title) && <p className="py-6 text-sm text-muted-foreground">{emptyText}</p>}
      <SortableContext items={headed.map((s) => headingId(s.heading as number))} strategy={verticalListSortingStrategy}>
      {visible.map((s) => (
        <SectionBlock key={s.key} section={s} lifted={activeHeading !== null && activeHeading === (typeof s.heading === "number" ? headingId(s.heading) : null)}>
          <SortableContext items={s.tasks.map((t) => t.id)} strategy={verticalListSortingStrategy}>
            {s.tasks.map((t) => {
              const isOver = overId === t.id && activeId !== null && activeId !== t.id;
              const activeSection = activeId !== null ? sectionOf(activeId) : undefined;
              const below = isOver && activeSection === s && s.tasks.findIndex((x) => x.id === activeId) < s.tasks.findIndex((x) => x.id === t.id);
              return (
                <TaskRow key={t.id} task={t} section={s.key} showProject={showProject} showWhen={showWhen}
                  insertion={isOver ? (below ? "below" : "above") : null} />
              );
            })}
          </SortableContext>
          {s.tasks.length === 0 && s.empty && <p className="py-2 pl-3 text-sm text-muted-foreground">{s.empty}</p>}
        </SectionBlock>
      ))}
      </SortableContext>
    </DndContext>
  );
}

function SectionBlock({ section, children, lifted }: { section: Section; children: ReactNode; lifted: boolean }) {
  const { setNodeRef: dropRef, isOver } = useDroppable({ id: `section:${section.key}`, data: { type: "section" } });
  const sortable = typeof section.heading === "number";
  // Sortable only for real headings; the loose "no heading" block stays first. The hook is
  // always called, and disabled for the loose block.
  const { attributes, listeners, setNodeRef: sortRef, setActivatorNodeRef, transform, transition } = useSortable({
    id: sortable ? headingId(section.heading as number) : `loose:${section.key}`,
    data: { type: "heading" },
    disabled: !sortable,
  });
  return (
    <section
      ref={(el) => { dropRef(el); sortRef(el); }}
      style={sortable ? { transform: CSS.Transform.toString(transform), transition } : undefined}
      className={`group/heading relative mb-6 rounded-xl ${lifted ? "opacity-40" : ""} ${isOver && section.tasks.length === 0 ? "bg-accent" : ""}`}
      aria-label={typeof section.title === "string" ? section.title : undefined}
    >
      {section.title && (
        <div {...(sortable ? pointerOnly(listeners) : {})} className="relative mb-1 flex items-center justify-between border-b border-border pb-1.5 pt-2">
          {sortable && (
            <Button
              ref={setActivatorNodeRef}
              {...attributes}
              {...listeners}
              variant="ghost"
              size="icon-xs"
              aria-label={`Reorder heading ${typeof section.title === "string" ? section.title : ""}`}
              className="absolute -left-7 top-2 hidden cursor-grab text-muted-foreground focus-visible:flex group-hover/heading:flex"
            >
              <GripVertical />
            </Button>
          )}
          <h2 className="text-sm font-semibold">{section.title}</h2>
          {section.actions}
        </div>
      )}
      {children}
    </section>
  );
}
