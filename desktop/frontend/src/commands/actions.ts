// Everything the UI can do, as plain functions over the API client and the UI
// store. Commands (the palette, shortcuts, menus) and components both call these,
// so an action behaves identically however it is triggered.
import { api, type NewTask, type Task } from "../api/client";
import { queryClient, keys } from "../state/queries";
import { useUI, LIST_ORDER, LIST_TITLES, viewKey, type View } from "../state/ui";
import { addDays } from "../lib/dates";

const ui = () => useUI.getState();

/** Run an API call; a refusal from the core (for example "task 3 has 2 open subtasks…") becomes a toast. */
export async function attempt<T>(fn: () => Promise<T>): Promise<T | undefined> {
  try {
    return await fn();
  } catch (err) {
    ui().flash(err instanceof Error ? err.message : String(err));
    return undefined;
  }
}

export function goTo(view: View): void {
  ui().setView(view);
  void api.getPrefs().then((p) => api.setPrefs({ ...p, lastView: viewKey(view) })).catch(() => undefined);
}

export const goList = (i: number) => goTo({ kind: "list", list: LIST_ORDER[i] });

export function selectedRow() {
  const { rows, selectedId } = ui();
  const i = rows.findIndex((r) => r.task.id === selectedId);
  return { rows, index: i, row: i >= 0 ? rows[i] : undefined };
}

export function moveSelection(delta: 1 | -1): void {
  const { rows, index } = selectedRow();
  if (rows.length === 0) return;
  const next = index < 0 ? (delta > 0 ? 0 : rows.length - 1) : Math.min(rows.length - 1, Math.max(0, index + delta));
  ui().select(rows[next].task.id);
  queueMicrotask(() => document.querySelector(`[data-task-id="${rows[next].task.id}"]`)?.scrollIntoView({ block: "nearest" }));
}

export async function toggleComplete(task: Task): Promise<void> {
  await attempt(() => (task.status === "open" ? api.complete(task.id) : api.reopen(task.id)));
}

export function toggleSelectedComplete(): void {
  const { row } = selectedRow();
  if (row) void toggleComplete(row.task);
}

export function toggleExpanded(): void {
  const { selectedId, expandedId } = ui();
  if (selectedId === null) return;
  ui().expand(expandedId === selectedId ? null : selectedId);
}

/** Move the selected task one place up or down within its section, as a single-row write. */
export async function reorderSelected(delta: -1 | 1): Promise<void> {
  const { rows, index, row } = selectedRow();
  if (!row) return;
  const sectionRows = rows.filter((r) => r.section === row.section);
  const i = sectionRows.findIndex((r) => r.task.id === row.task.id);
  const j = i + delta;
  if (j < 0 || j >= sectionRows.length) return;
  // Neighbours in the order the task will end up in.
  const without = sectionRows.filter((_, k) => k !== i);
  const after = without[j - 1]?.task.id ?? null;
  const before = without[j]?.task.id ?? null;
  void index;
  await attempt(() => api.placeTask(row.task.id, { afterId: after, beforeId: before }));
}

export function deleteSelected(): void {
  const { row } = selectedRow();
  if (!row) return;
  ui().openDialog({
    kind: "confirm",
    title: "Delete task",
    message: `Delete “${row.task.title}” and its subtasks? This can’t be undone.`,
    confirm: "Delete",
    danger: true,
    onConfirm: () => {
      ui().select(null);
      ui().expand(null);
      void attempt(() => api.deleteTask(row.task.id));
    },
  });
}

/** Where a new task lands depends on the view it is created in. */
export function newTaskDefaults(view: View, today: string): Partial<NewTask> {
  if (view.kind === "project") return { projectId: view.id };
  if (view.kind === "area") return { areaId: view.id };
  switch (view.list) {
    case "today": return { whenKind: "scheduled", whenDate: today };
    case "upcoming": return { whenKind: "scheduled", whenDate: addDays(today, 1) };
    case "someday": return { whenKind: "someday" };
    default: return {};
  }
}

export async function createTask(title: string, view: View, today: string, extra: Partial<NewTask> = {}): Promise<Task | undefined> {
  const t = title.trim();
  if (!t) return undefined;
  const created = await attempt(() => api.addTask({
    title: t, notes: "", whenKind: "", whenDate: "", deadline: "", evening: false,
    projectId: null, areaId: null, headingId: null, parentId: null, tags: [],
    ...newTaskDefaults(view, today), ...extra,
  }));
  if (created) ui().select(created.id);
  return created;
}

export function promptName(opts: { title: string; label: string; initial?: string; confirm: string }, onSubmit: (v: string) => void): void {
  ui().openDialog({ kind: "prompt", ...opts, onSubmit });
}

export function newProject(areaId: number | null = null): void {
  promptName({ title: "New project", label: "Project name", confirm: "Create" }, async (name) => {
    const p = await attempt(() => api.addProject(name, areaId));
    if (p) goTo({ kind: "project", id: p.id });
  });
}

export function newArea(): void {
  promptName({ title: "New area", label: "Area name", confirm: "Create" }, async (name) => {
    const a = await attempt(() => api.addArea(name));
    if (a) goTo({ kind: "area", id: a.id });
  });
}

export function newHeading(projectId: number): void {
  promptName({ title: "New heading", label: "Heading", confirm: "Add" }, (title) => void attempt(() => api.addHeading(projectId, title)));
}

export function cycleTheme(): void {
  const order = ["system", "light", "dark"] as const;
  const next = order[(order.indexOf(ui().theme) + 1) % order.length];
  void setTheme(next);
}

export async function setTheme(theme: "system" | "light" | "dark"): Promise<void> {
  ui().setTheme(theme);
  const p = await api.getPrefs();
  await api.setPrefs({ ...p, theme });
}

export function refreshAll(): void {
  void queryClient.invalidateQueries();
}

export { keys, LIST_TITLES };
