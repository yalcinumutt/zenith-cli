import {
  DndContext, KeyboardSensor, PointerSensor, closestCenter, useDroppable, useSensor, useSensors,
  type DragEndEvent, type DragOverEvent, type DragStartEvent,
} from "@dnd-kit/core";
import { SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from "@dnd-kit/sortable";
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

  // Publish what is on screen so keyboard navigation and Mod+Up/Down know the order.
  const rows: Row[] = useMemo(() => sections.flatMap((s) => s.tasks.map((task) => ({ task, section: s.key }))), [sections]);
  useEffect(() => { useUI.getState().setRows(rows); return () => useUI.getState().setRows([]); }, [rows]);

  const sectionOf = (taskId: number) => sections.find((s) => s.tasks.some((t) => t.id === taskId));

  function handleEnd(e: DragEndEvent) {
    setOverId(null);
    setActiveId(null);
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
  const handleStart = (e: DragStartEvent) => setActiveId(Number(e.active.id));

  const total = sections.reduce((n, s) => n + s.tasks.length, 0);
  const visible = sections.filter((s) => s.tasks.length > 0 || s.empty || s.title);

  return (
    <DndContext sensors={sensors} collisionDetection={closestCenter} onDragStart={handleStart} onDragOver={handleOver} onDragEnd={handleEnd} onDragCancel={() => { setOverId(null); setActiveId(null); }}>
      {total === 0 && sections.every((s) => !s.title) && <p className="py-6 text-sm text-muted">{emptyText}</p>}
      {visible.map((s) => (
        <SectionBlock key={s.key} section={s}>
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
          {s.tasks.length === 0 && s.empty && <p className="py-2 pl-3 text-sm text-muted">{s.empty}</p>}
        </SectionBlock>
      ))}
    </DndContext>
  );
}

function SectionBlock({ section, children }: { section: Section; children: ReactNode }) {
  const { setNodeRef, isOver } = useDroppable({ id: `section:${section.key}` });
  return (
    <section ref={setNodeRef} className={`mb-6 rounded-xl ${isOver && section.tasks.length === 0 ? "bg-selected" : ""}`} aria-label={typeof section.title === "string" ? section.title : undefined}>
      {section.title && (
        <div className="mb-1 flex items-center justify-between border-b border-line pb-1.5 pt-2">
          <h2 className="text-[13px] font-semibold">{section.title}</h2>
          {section.actions}
        </div>
      )}
      {children}
    </section>
  );
}
