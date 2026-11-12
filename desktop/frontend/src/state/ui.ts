import { toast } from "sonner";
import { create } from "zustand";
import type { Task } from "../api/client";

export type ListId = "inbox" | "today" | "upcoming" | "anytime" | "someday" | "logbook";
export const LIST_ORDER: ListId[] = ["inbox", "today", "upcoming", "anytime", "someday", "logbook"];
export const LIST_TITLES: Record<ListId, string> = {
  inbox: "Inbox", today: "Today", upcoming: "Upcoming", anytime: "Anytime", someday: "Someday", logbook: "Logbook",
};

/** What the main pane shows. A typed value, not a URL: the app has no router. */
export type View =
  | { kind: "list"; list: ListId }
  | { kind: "project"; id: number }
  | { kind: "area"; id: number }
  | { kind: "habits" };

export function viewKey(v: View): string {
  if (v.kind === "habits") return "habits";
  return v.kind === "list" ? v.list : `${v.kind}:${v.id}`;
}

export function parseView(key: string | undefined): View {
  if (key && (LIST_ORDER as string[]).includes(key)) return { kind: "list", list: key as ListId };
  if (key === "habits") return { kind: "habits" };
  const m = /^(project|area):(\d+)$/.exec(key ?? "");
  if (m) return { kind: m[1] as "project" | "area", id: Number(m[2]) };
  return { kind: "list", list: "today" };
}

export type Theme = "system" | "light" | "dark";

export type Dialog =
  | { kind: "prompt"; title: string; label: string; initial?: string; confirm: string; onSubmit: (value: string) => void }
  | { kind: "confirm"; title: string; message: string; confirm: string; danger?: boolean; onConfirm: () => void }
  | { kind: "date"; title: string; initial: string; confirm: string; onSubmit: (date: string) => void }
  | { kind: "about" };

/** A row currently on screen, in display order, with the section it sits in. */
export interface Row {
  task: Task;
  section: string;
}

interface UIState {
  moveTaskId: number | null;
  setMoveTask: (id: number | null) => void;
  view: View;
  setView: (v: View) => void;
  selectedId: number | null;
  select: (id: number | null) => void;
  expandedId: number | null;
  expand: (id: number | null) => void;
  /** Rows on screen, published by the active view for keyboard navigation and reordering. */
  rows: Row[];
  setRows: (rows: Row[]) => void;
  theme: Theme;
  setTheme: (t: Theme) => void;
  paletteOpen: boolean;
  setPalette: (open: boolean) => void;
  newTaskFocus: number;
  focusNewTask: () => void;
  reminderFocus: number;
  focusReminder: () => void;
  dialog: Dialog | null;
  openDialog: (d: Dialog | null) => void;
  flash: (text: string) => void;
}

export const useUI = create<UIState>((set) => ({
  moveTaskId: null,
  setMoveTask: (moveTaskId) => set({ moveTaskId }),
  view: { kind: "list", list: "today" },
  setView: (view) => set({ view, selectedId: null, expandedId: null }),
  selectedId: null,
  select: (selectedId) => set({ selectedId }),
  expandedId: null,
  expand: (expandedId) => set(expandedId === null ? { expandedId } : { expandedId, selectedId: expandedId }),
  rows: [],
  setRows: (rows) => set({ rows }),
  theme: "system",
  setTheme: (theme) => set({ theme }),
  paletteOpen: false,
  setPalette: (paletteOpen) => set({ paletteOpen }),
  newTaskFocus: 0,
  focusNewTask: () => set((s) => ({ newTaskFocus: s.newTaskFocus + 1 })),
  reminderFocus: 0,
  focusReminder: () => set((s) => ({ reminderFocus: s.reminderFocus + 1 })),
  dialog: null,
  openDialog: (dialog) => set({ dialog }),
  flash: (text) => void toast(text),
}));
