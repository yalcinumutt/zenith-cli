import { BellIcon, CalendarDaysIcon, CalendarIcon, CheckIcon, FolderInputIcon, LayersIcon, MoonIcon, PencilIcon, StarIcon, SunriseIcon, SunsetIcon, Trash2Icon, XIcon, type LucideIcon } from "lucide-react";
import type { ReactNode } from "react";
import {
  ContextMenu, ContextMenuContent, ContextMenuItem, ContextMenuLabel, ContextMenuSeparator, ContextMenuShortcut, ContextMenuTrigger,
} from "@/components/ui/context-menu";
import { api, type Task } from "../api/client";
import { attempt, deleteTask, pickDateFor, scheduleTask, toggleComplete, type WhenChoice } from "../commands/actions";
import { formatShortcut } from "../lib/platform";
import { useUI } from "../state/ui";

/** Run after the menu has closed, so a dialog opened by the action isn't fighting for focus. */
const later = (fn: () => void) => () => void setTimeout(fn, 0);

/**
 * Wraps a task row with its right-click menu. The same menu opens from the keyboard
 * (`w`, or the menu key) by sending the row a contextmenu event at its position.
 * Each item's letter is a shortcut while the menu is open.
 */
export function TaskContextMenu({ task, children }: { task: Task; children: ReactNode }) {
  const open = task.status === "open";
  const when = (label: string, Icon: LucideIcon, key: string, choice: WhenChoice) => (
    <Item label={label} Icon={Icon} accel={key} onSelect={later(() => void scheduleTask(task, choice))} />
  );

  function onKeyDown(e: React.KeyboardEvent<HTMLDivElement>) {
    if (e.metaKey || e.ctrlKey || e.altKey) return;
    const hit = e.currentTarget.querySelector<HTMLElement>(`[data-accel="${e.key.toLowerCase()}"]`);
    if (hit) { e.preventDefault(); hit.click(); }
  }

  return (
    <ContextMenu>
      <ContextMenuTrigger asChild>{children}</ContextMenuTrigger>
      <ContextMenuContent className="w-56" onKeyDown={onKeyDown} aria-label={`Actions for ${task.title}`}>
        <Item label={open ? "Complete" : "Reopen"} Icon={CheckIcon} accel="x" onSelect={later(() => void toggleComplete(task))} />
        <Item label="Edit…" Icon={PencilIcon} onSelect={later(() => useUI.getState().expand(task.id))} />
        <ContextMenuSeparator />
        <ContextMenuLabel>When</ContextMenuLabel>
        {when("Today", StarIcon, "t", "today")}
        {when("This Evening", SunsetIcon, "e", "evening")}
        {when("Tomorrow", SunriseIcon, "m", "tomorrow")}
        {when("Next Monday", CalendarDaysIcon, "n", "nextweek")}
        <Item label="Pick a date…" Icon={CalendarIcon} accel="d" onSelect={later(() => pickDateFor(task))} />
        {when("Someday", MoonIcon, "s", "someday")}
        {when("Anytime", LayersIcon, "a", "anytime")}
        <ContextMenuSeparator />
        <Item label="Move to…" Icon={FolderInputIcon} accel="p" onSelect={later(() => useUI.getState().setMoveTask(task.id))} />
        <Item label="Add a reminder…" Icon={BellIcon} onSelect={later(() => { useUI.getState().expand(task.id); useUI.getState().focusReminder(); })} />
        <ContextMenuSeparator />
        {open && <Item label="Cancel task" Icon={XIcon} onSelect={later(() => void attempt(() => api.cancel(task.id)))} />}
        <Item label="Delete…" Icon={Trash2Icon} accel="backspace" destructive onSelect={later(() => deleteTask(task))} />
      </ContextMenuContent>
    </ContextMenu>
  );
}

function Item({ label, Icon, accel, destructive, onSelect }: { label: string; Icon: LucideIcon; accel?: string; destructive?: boolean; onSelect: () => void }) {
  return (
    <ContextMenuItem variant={destructive ? "destructive" : "default"} data-accel={accel} onSelect={onSelect}>
      <Icon />
      {label}
      {accel && <ContextMenuShortcut>{formatShortcut(accel)}</ContextMenuShortcut>}
    </ContextMenuItem>
  );
}
