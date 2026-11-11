// The single command registry. The shortcut handler, the command palette and
// menus are all generated from this list: adding a command means adding one entry.
import { api } from "../api/client";
import { useUI, LIST_ORDER, LIST_TITLES, type View } from "../state/ui";
import {
  attempt, cycleTheme, deleteSelected, goList, goTo, moveSelection, newArea, newHeading, newProject,
  promptName, refreshAll, reorderSelected, selectedRow, toggleExpanded, toggleSelectedComplete,
} from "./actions";
import { addDays } from "../lib/dates";
import { queryClient, keys } from "../state/queries";

export interface Command {
  id: string;
  title: string;
  group: "Navigate" | "Task" | "Organise" | "App";
  /** e.g. "mod+k", "x", "mod+arrowup". `mod` is Command on macOS, Control elsewhere. */
  shortcut?: string;
  /** Extra words the palette matches on. */
  keywords?: string[];
  /** Hidden from the palette and ignored by the shortcut when this returns false. */
  when?: () => boolean;
  /** Plain-key shortcuts (no modifier) don't fire while typing in a field. */
  run: () => void;
}

const ui = () => useUI.getState();
const hasSelection = () => selectedRow().row !== undefined;
const inProject = () => ui().view.kind === "project";
const currentView = (): View => ui().view;

async function today(): Promise<string> {
  return (await queryClient.fetchQuery({ queryKey: keys.info, queryFn: api.info, staleTime: 10_000 })).today;
}

function schedule(kind: "today" | "tomorrow" | "someday" | "anytime"): void {
  const { row } = selectedRow();
  if (!row) return;
  void (async () => {
    const t = await today();
    const patch =
      kind === "today" ? { whenKind: "scheduled", whenDate: t } :
      kind === "tomorrow" ? { whenKind: "scheduled", whenDate: addDays(t, 1) } :
      { whenKind: kind, whenDate: "" };
    await attempt(() => api.updateTask(row.task.id, patch));
  })();
}

export const commands: Command[] = [
  // ---- navigate
  ...LIST_ORDER.map<Command>((list, i) => ({
    id: `go-${list}`,
    title: `Go to ${LIST_TITLES[list]}`,
    group: "Navigate",
    shortcut: `mod+${i + 1}`,
    keywords: ["list", "open", "show"],
    run: () => goList(i),
  })),
  { id: "select-next", title: "Select next task", group: "Navigate", shortcut: "arrowdown", when: () => ui().rows.length > 0, run: () => moveSelection(1) },
  { id: "select-prev", title: "Select previous task", group: "Navigate", shortcut: "arrowup", when: () => ui().rows.length > 0, run: () => moveSelection(-1) },
  { id: "palette", title: "Open command palette", group: "Navigate", shortcut: "mod+k", run: () => ui().setPalette(true) },

  // ---- task
  { id: "new-task", title: "New to-do", group: "Task", shortcut: "mod+n", keywords: ["add", "create"], run: () => ui().focusNewTask() },
  { id: "toggle-expand", title: "Expand or collapse task", group: "Task", shortcut: "space", when: hasSelection, run: toggleExpanded },
  { id: "open-task", title: "Open task to edit", group: "Task", shortcut: "enter", when: hasSelection, run: toggleExpanded },
  { id: "close-task", title: "Close open task", group: "Task", shortcut: "escape", when: () => ui().expandedId !== null, run: () => ui().expand(null) },
  { id: "complete", title: "Complete or reopen task", group: "Task", shortcut: "x", keywords: ["done", "check"], when: hasSelection, run: toggleSelectedComplete },
  { id: "move-up", title: "Move task up", group: "Task", shortcut: "mod+arrowup", keywords: ["reorder"], when: hasSelection, run: () => void reorderSelected(-1) },
  { id: "move-down", title: "Move task down", group: "Task", shortcut: "mod+arrowdown", keywords: ["reorder"], when: hasSelection, run: () => void reorderSelected(1) },
  { id: "delete-task", title: "Delete task", group: "Task", shortcut: "mod+backspace", when: hasSelection, run: deleteSelected },
  { id: "cancel-task", title: "Cancel task", group: "Task", keywords: ["drop"], when: hasSelection, run: () => { const { row } = selectedRow(); if (row) void attempt(() => api.cancel(row.task.id)); } },
  { id: "when-today", title: "Schedule for today", group: "Task", keywords: ["when", "plan"], when: hasSelection, run: () => schedule("today") },
  { id: "when-tomorrow", title: "Schedule for tomorrow", group: "Task", keywords: ["when", "plan"], when: hasSelection, run: () => schedule("tomorrow") },
  { id: "when-someday", title: "Move to Someday", group: "Task", keywords: ["when", "park"], when: hasSelection, run: () => schedule("someday") },
  { id: "when-anytime", title: "Move to Anytime (unscheduled)", group: "Task", keywords: ["when", "clear"], when: hasSelection, run: () => schedule("anytime") },
  { id: "remind", title: "Add a reminder to task", group: "Task", keywords: ["notify", "alarm"], when: hasSelection, run: () => { const { row } = selectedRow(); if (row) { ui().expand(row.task.id); ui().focusReminder(); } } },

  // ---- organise
  { id: "new-area", title: "New area", group: "Organise", run: newArea },
  { id: "new-project", title: "New project", group: "Organise", run: () => { const v = currentView(); newProject(v.kind === "area" ? v.id : null); } },
  { id: "new-heading", title: "New heading in this project", group: "Organise", when: inProject, run: () => { const v = currentView(); if (v.kind === "project") newHeading(v.id); } },
  {
    id: "rename-container", title: "Rename this project or area", group: "Organise",
    when: () => ui().view.kind !== "list",
    run: () => {
      const v = currentView();
      if (v.kind === "list") return;
      promptName({ title: "Rename", label: "Name", confirm: "Rename" }, (name) => void attempt(() => (v.kind === "project" ? api.renameProject(v.id, name) : api.renameArea(v.id, name))));
    },
  },
  { id: "complete-project", title: "Complete this project", group: "Organise", when: inProject, run: () => { const v = currentView(); if (v.kind === "project") void attempt(async () => { await api.completeProject(v.id); goTo({ kind: "list", list: "today" }); }); } },
  {
    id: "delete-container", title: "Delete this project or area", group: "Organise",
    when: () => ui().view.kind !== "list",
    run: () => {
      const v = currentView();
      if (v.kind === "list") return;
      ui().openDialog({
        kind: "confirm", title: `Delete ${v.kind}`, confirm: "Delete", danger: true,
        message: v.kind === "project" ? "This deletes the project with its headings and every task in it." : "The area is removed; its projects become standalone and its tasks unfiled.",
        onConfirm: () => void attempt(async () => { await (v.kind === "project" ? api.deleteProject(v.id) : api.deleteArea(v.id)); goTo({ kind: "list", list: "today" }); }),
      });
    },
  },

  // ---- app
  { id: "theme", title: "Change theme (system / light / dark)", group: "App", keywords: ["dark", "light", "appearance"], run: cycleTheme },
  { id: "refresh", title: "Refresh", group: "App", keywords: ["reload"], run: refreshAll },
  { id: "about", title: "About Zenith", group: "App", keywords: ["beta", "sponsor", "version"], run: () => ui().openDialog({ kind: "about" }) },
];
