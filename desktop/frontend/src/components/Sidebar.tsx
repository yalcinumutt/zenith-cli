import {
  DndContext, KeyboardSensor, PointerSensor, closestCenter, useDroppable, useSensor, useSensors,
  type CollisionDetection, type DragEndEvent, type DragOverEvent, type DragStartEvent,
} from "@dnd-kit/core";
import { SortableContext, sortableKeyboardCoordinates, useSortable, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { CalendarClockIcon, CalendarIcon, CheckCheckIcon, ChevronRightIcon, FlameIcon, FolderOpenIcon, InboxIcon, InfoIcon, LayersIcon, MoonIcon, PencilIcon, PlusIcon, StarIcon, SunMoonIcon, type LucideIcon } from "lucide-react";
import { useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { ContextMenu, ContextMenuContent, ContextMenuItem, ContextMenuTrigger } from "@/components/ui/context-menu";
import {
  Sidebar as ShadSidebar, SidebarContent, SidebarFooter, SidebarGroup, SidebarGroupLabel, SidebarMenu, SidebarMenuBadge,
  SidebarMenuButton, SidebarMenuItem, SidebarProvider, SidebarSeparator,
} from "@/components/ui/sidebar";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { cn, pointerOnly } from "@/lib/utils";
import { api, type Area, type Project } from "../api/client";
import { attempt, cycleTheme, goTo, newArea, newProject, promptName } from "../commands/actions";
import { useCounts, useSidebar } from "../state/queries";
import { LIST_ORDER, LIST_TITLES, useUI, type ListId, type View } from "../state/ui";

const ICONS: Record<ListId, LucideIcon> = { inbox: InboxIcon, today: StarIcon, upcoming: CalendarIcon, anytime: LayersIcon, someday: MoonIcon, logbook: CheckCheckIcon };
const ICON_TONE: Record<ListId, string> = { inbox: "text-primary", today: "text-amber-600 dark:text-amber-400", upcoming: "text-primary", anytime: "text-primary", someday: "text-muted-foreground", logbook: "text-muted-foreground" };

function same(a: View, b: View): boolean {
  if (a.kind !== b.kind) return false;
  if (a.kind === "list" && b.kind === "list") return a.list === b.list;
  if (a.kind === "habits" || b.kind === "habits" || a.kind === "plan" || b.kind === "plan") return a.kind === b.kind;
  return a.kind !== "list" && b.kind !== "list" && a.id === b.id;
}

type DragKind = "project" | "area";
const pid = (id: number) => `p:${id}`;
const aid = (id: number) => `a:${id}`;
const cid = (areaId: number | null) => `c:${areaId ?? "none"}`;

/** A navigation row: a list, project or area. */
function NavRow({ target, label, icon: Icon, tone, count }: { target: View; label: string; icon: LucideIcon; tone?: string; count?: number }) {
  const active = useUI((s) => same(s.view, target));
  return (
    <SidebarMenuButton isActive={active} aria-current={active ? "page" : undefined} onClick={() => goTo(target)}>
      <Icon className={tone ?? "text-primary"} aria-hidden />
      <span>{label}</span>
      {!!count && <SidebarMenuBadge aria-label={`${count} tasks`}>{count}</SidebarMenuBadge>}
    </SidebarMenuButton>
  );
}

/** A project row that can be dragged to reorder it or to move it into another area. */
function ProjectItem({ project, containerId, insertion }: { project: Project; containerId: number | null; insertion: "above" | "below" | null }) {
  const { listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: pid(project.id), data: { type: "project", containerId } });
  return (
    <SidebarMenuItem
      ref={setNodeRef}
      {...pointerOnly(listeners)}
      style={{ transform: CSS.Transform.toString(transform), transition }}
      className={cn(
        isDragging && "opacity-40",
        insertion === "above" && "before:absolute before:-top-px before:left-2 before:right-2 before:h-0.5 before:bg-primary",
        insertion === "below" && "after:absolute after:-bottom-px after:left-2 after:right-2 after:h-0.5 after:bg-primary",
      )}
    >
      <NavRow target={{ kind: "project", id: project.id }} label={project.name} icon={FolderOpenIcon} count={project.openTasks} />
    </SidebarMenuItem>
  );
}

interface DragState { overId: string | null; activeId: string | null; dragging: DragKind | null; overBelow: boolean }

/** The project list of one area (or the standalone list); also the drop target for an empty one. */
function ProjectList({ areaId, projects, drag }: { areaId: number | null; projects: Project[]; drag: DragState }) {
  const { overId, activeId, dragging, overBelow } = drag;
  const { setNodeRef, isOver } = useDroppable({ id: cid(areaId), data: { type: "project-container" } });
  const activeIndex = projects.findIndex((p) => pid(p.id) === activeId);
  return (
    <SidebarMenu ref={setNodeRef} className={cn("min-h-1 rounded-md", dragging === "project" && projects.length === 0 && "min-h-8 border border-dashed", isOver && "bg-accent")}>
      <SortableContext items={projects.map((p) => pid(p.id))} strategy={verticalListSortingStrategy}>
        {projects.map((p, i) => {
          const hot = overId === pid(p.id) && activeId !== null && activeId !== pid(p.id);
          const below = hot && (activeIndex >= 0 ? activeIndex < i : overBelow); // within a list the move direction decides; from another list, the pointer side
          return <ProjectItem key={p.id} project={p} containerId={areaId} insertion={hot ? (below ? "below" : "above") : null} />;
        })}
      </SortableContext>
      {dragging === "project" && projects.length === 0 && <p className="px-2 py-1.5 text-xs text-muted-foreground">Drop a project here</p>}
    </SidebarMenu>
  );
}

/** An area block: a draggable header (reorders areas) over its project list (a drop target for projects). */
function AreaBlock({ area, drag }: { area: Area; drag: DragState }) {
  const { listeners, setNodeRef, transform, transition, isDragging } =
    useSortable({ id: aid(area.id), data: { type: "area" } });
  const active = useUI((s) => s.view.kind === "area" && s.view.id === area.id);
  const collapsed = useUI((s) => s.collapsedAreas.includes(area.id));
  // Keep the list open while a project is dragged over it, so it can be dropped in.
  const open = !collapsed || drag.dragging === "project";
  const toggle = () => useUI.getState().setAreaCollapsed(area.id, !collapsed);
  const later = (fn: () => void) => () => void setTimeout(fn, 0);
  return (
    <SidebarGroup ref={setNodeRef} style={{ transform: CSS.Transform.toString(transform), transition }} className={cn("p-0 pt-4", isDragging && "opacity-40")}>
      <ContextMenu>
        <ContextMenuTrigger asChild>
          <div className="group/menu-item relative flex items-center" {...pointerOnly(listeners)}>
            <Button variant="ghost" size="icon-xs" className="shrink-0 text-muted-foreground" aria-expanded={open} aria-label={`${open ? "Collapse" : "Expand"} area ${area.name}`} onClick={toggle}>
              <ChevronRightIcon className={cn("transition-transform", open && "rotate-90")} />
            </Button>
            <button
              onClick={() => goTo({ kind: "area", id: area.id })}
              aria-current={active ? "page" : undefined}
              className={cn(
                "h-7 min-w-0 flex-1 cursor-pointer truncate rounded-md px-1.5 text-left text-xs font-semibold uppercase tracking-wide outline-none hover:bg-sidebar-accent hover:text-sidebar-accent-foreground focus-visible:ring-2 focus-visible:ring-sidebar-ring",
                active ? "bg-sidebar-accent text-sidebar-accent-foreground" : "text-sidebar-foreground/70",
              )}
            >
              {area.name}
            </button>
          </div>
        </ContextMenuTrigger>
        <ContextMenuContent className="w-48" aria-label={`Actions for ${area.name}`}>
          <ContextMenuItem onSelect={later(() => { useUI.getState().setAreaCollapsed(area.id, false); newProject(area.id); })}><PlusIcon />New project</ContextMenuItem>
          <ContextMenuItem onSelect={later(() => promptName({ title: "Rename area", label: "Area name", initial: area.name, confirm: "Rename" }, (name) => void attempt(() => api.renameArea(area.id, name))))}><PencilIcon />Rename area</ContextMenuItem>
        </ContextMenuContent>
      </ContextMenu>
      {open && <ProjectList areaId={area.id} projects={area.projects} drag={drag} />}
    </SidebarGroup>
  );
}

// Projects only collide with projects and project lists; areas only with areas.
const collide: CollisionDetection = (args) => {
  const kind = args.active.data.current?.type as DragKind | undefined;
  const want = kind === "area" ? ["area"] : ["project", "project-container"];
  return closestCenter({ ...args, droppableContainers: args.droppableContainers.filter((c) => want.includes(c.data.current?.type)) });
};

const parseP = (id: string) => Number(id.slice(2));

export function Sidebar() {
  const counts = useCounts().data;
  const sidebar = useSidebar().data;
  const theme = useUI((s) => s.theme);
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 6 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const [dragging, setDragging] = useState<DragKind | null>(null);
  const [activeId, setActiveId] = useState<string | null>(null);
  const [overId, setOverId] = useState<string | null>(null);
  const [overBelow, setOverBelow] = useState(false);
  const reset = () => { setDragging(null); setActiveId(null); setOverId(null); setOverBelow(false); };

  const areas = sidebar?.areas ?? [];
  const standalone = sidebar?.standalone ?? [];
  const listOf = (containerId: number | null): Project[] => containerId === null ? standalone : areas.find((a) => a.id === containerId)?.projects ?? [];
  const containerOfProject = (id: number): number | null => areas.find((a) => a.projects.some((p) => p.id === id))?.id ?? null;

  function onEnd(e: DragEndEvent) {
    const below = overBelow;
    const active = String(e.active.id);
    const over = e.over ? String(e.over.id) : null;
    reset();
    if (!over || over === active) return;

    if (active.startsWith("a:") && over.startsWith("a:")) {
      const id = parseP(active);
      const ordered = areas.filter((a) => a.id !== id);
      const overIndex = ordered.findIndex((a) => a.id === parseP(over));
      const movingDown = areas.findIndex((a) => a.id === id) < areas.findIndex((a) => a.id === parseP(over));
      const slot = movingDown ? overIndex + 1 : overIndex;
      void attempt(() => api.reorderArea(id, ordered[slot - 1]?.id ?? null, ordered[slot]?.id ?? null));
      return;
    }
    if (!active.startsWith("p:")) return;

    const id = parseP(active);
    const from = containerOfProject(id);
    let to: number | null;
    let after: number | null;
    let before: number | null = null;
    if (over.startsWith("c:")) {
      to = over === cid(null) ? null : Number(over.slice(2));
      const ordered = listOf(to).filter((p) => p.id !== id);
      after = ordered[ordered.length - 1]?.id ?? null; // dropped on a list: goes to its end
    } else {
      const overProject = parseP(over);
      to = containerOfProject(overProject);
      const ordered = listOf(to).filter((p) => p.id !== id);
      const overIndex = ordered.findIndex((p) => p.id === overProject);
      const movingDown = from === to ? listOf(from).findIndex((p) => p.id === id) < listOf(to).findIndex((p) => p.id === overProject) : below;
      const slot = movingDown ? overIndex + 1 : overIndex;
      after = ordered[slot - 1]?.id ?? null;
      before = ordered[slot]?.id ?? null;
    }
    void attempt(() => api.placeProject(id, from === to ? { afterId: after, beforeId: before } : { areaId: to, afterId: after, beforeId: before }));
  }

  const onStart = (e: DragStartEvent) => { setActiveId(String(e.active.id)); setDragging(e.active.data.current?.type as DragKind); };
  const onOver = (e: DragOverEvent) => {
    setOverId(e.over ? String(e.over.id) : null);
    const moved = e.active.rect.current.translated;
    setOverBelow(!!e.over && !!moved && moved.top + moved.height / 2 > e.over.rect.top + e.over.rect.height / 2);
  };

  const drag: DragState = { overId, activeId, dragging, overBelow };

  return (
    <SidebarProvider className="min-h-0 w-auto" style={{ "--sidebar-width": "15rem" } as React.CSSProperties}>
      <ShadSidebar collapsible="none" className="border-r" aria-label="Sidebar">
        <SidebarContent className="p-3 pt-6">
          <SidebarMenu aria-label="Lists">
            {LIST_ORDER.map((l) => (
              <SidebarMenuItem key={l}>
                <NavRow target={{ kind: "list", list: l }} label={LIST_TITLES[l]} icon={ICONS[l]} tone={ICON_TONE[l]}
                  count={l === "inbox" ? counts?.inbox : l === "today" ? counts?.today : l === "upcoming" ? counts?.upcoming : undefined} />
              </SidebarMenuItem>
            ))}
          </SidebarMenu>

          <SidebarMenu className="pt-4">
            <SidebarMenuItem>
              <NavRow target={{ kind: "plan" }} label="Daily planning" icon={CalendarClockIcon} />
            </SidebarMenuItem>
            <SidebarMenuItem>
              <NavRow target={{ kind: "habits" }} label="Habits" icon={FlameIcon} tone="text-amber-600 dark:text-amber-400" />
            </SidebarMenuItem>
          </SidebarMenu>

          <DndContext sensors={sensors} collisionDetection={collide} onDragStart={onStart} onDragOver={onOver} onDragEnd={onEnd} onDragCancel={reset}>
            <SortableContext items={areas.map((a) => aid(a.id))} strategy={verticalListSortingStrategy}>
              {areas.map((a) => <AreaBlock key={a.id} area={a} drag={drag} />)}
            </SortableContext>
            {(standalone.length > 0 || dragging === "project") && (
              <SidebarGroup className="p-0 pt-4">
                <SidebarGroupLabel className="h-7 font-semibold uppercase tracking-wide">Projects</SidebarGroupLabel>
                <ProjectList areaId={null} projects={standalone} drag={drag} />
              </SidebarGroup>
            )}
          </DndContext>
        </SidebarContent>

        <SidebarSeparator className="mx-0" />
        <SidebarFooter className="p-3">
          <SidebarMenu>
            <SidebarMenuItem>
              <SidebarMenuButton className="text-muted-foreground" onClick={() => newProject()}><PlusIcon aria-hidden /><span>New project</span></SidebarMenuButton>
            </SidebarMenuItem>
            <SidebarMenuItem>
              <SidebarMenuButton className="text-muted-foreground" onClick={newArea}><PlusIcon aria-hidden /><span>New area</span></SidebarMenuButton>
            </SidebarMenuItem>
          </SidebarMenu>
          <div className="flex items-center justify-between pt-1">
            <Button variant="ghost" size="sm" className="text-muted-foreground" onClick={() => useUI.getState().openDialog({ kind: "about" })}>
              <InfoIcon data-icon="inline-start" />About<Badge variant="secondary" className="ml-1 uppercase">Beta</Badge>
            </Button>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button variant="ghost" size="icon-sm" className="text-muted-foreground" onClick={cycleTheme} aria-label={`Theme: ${theme}. Change theme`}><SunMoonIcon /></Button>
              </TooltipTrigger>
              <TooltipContent>Theme: {theme}</TooltipContent>
            </Tooltip>
          </div>
        </SidebarFooter>
      </ShadSidebar>
    </SidebarProvider>
  );
}
