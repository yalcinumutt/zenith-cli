// The one module that talks to Go. It is the only importer of the generated
// bindings (enforced by lint): everything else in the app uses this file's
// hand-written, null-free types and functions (ADR 0001, ADR 0007).
import { Events } from "@wailsio/runtime";
import * as Svc from "../../bindings/github.com/yalcinumutt/zenith-cli/internal/appapi/service.js";
import type {
  AreaDTO,
  ChangeEvent,
  Counts,
  HabitDTO,
  HeadingDTO,
  Info,
  NewTask,
  Place,
  Prefs,
  ProjectDTO,
  ReminderDTO,
  RepeatDTO,
  TagDTO,
  TaskDTO,
  TaskPatch,
} from "../../bindings/github.com/yalcinumutt/zenith-cli/internal/appapi/index.js";

export type { Counts, HeadingDTO as Heading, Info, NewTask, Place, Prefs, ReminderDTO as Reminder, RepeatDTO as Repeat, TagDTO as Tag, TaskPatch, ChangeEvent };

/** A habit with the days it was done (YYYY-MM-DD, oldest first). */
export type Habit = Omit<HabitDTO, "days"> & { days: string[] };
export type Task = Omit<TaskDTO, "tags"> & { tags: TagDTO[] };
export type Project = ProjectDTO;
export type Area = Omit<AreaDTO, "projects"> & { projects: ProjectDTO[] };
export interface Today { day: Task[]; evening: Task[] }
export interface Sidebar { areas: Area[]; standalone: Project[] }
export interface ProjectView { project: Project; headings: HeadingDTO[]; tasks: Task[] }
export interface AreaView { area: Area; tasks: Task[] }

const task = (t: TaskDTO): Task => ({ ...t, tags: t.tags ?? [] });
const tasks = (ts: TaskDTO[] | null): Task[] => (ts ?? []).map(task);
const area = (a: AreaDTO): Area => ({ ...a, projects: a.projects ?? [] });

/** The asynchronous API. Errors reject with the core's message as `Error.message`. */
export const api = {
  info: (): Promise<Info> => Svc.Info(),
  counts: (): Promise<Counts> => Svc.Counts(),

  today: async (): Promise<Today> => {
    const t = await Svc.Today();
    return { day: tasks(t.day), evening: tasks(t.evening) };
  },
  list: async (kind: string): Promise<Task[]> => tasks(await Svc.List(kind)),
  sidebar: async (): Promise<Sidebar> => {
    const s = await Svc.Sidebar();
    return { areas: (s.areas ?? []).map(area), standalone: s.standalone ?? [] };
  },
  projectView: async (id: number): Promise<ProjectView> => {
    const v = await Svc.ProjectView(id);
    return { project: v.project, headings: v.headings ?? [], tasks: tasks(v.tasks) };
  },
  areaView: async (id: number): Promise<AreaView> => {
    const v = await Svc.AreaView(id);
    return { area: area(v.area), tasks: tasks(v.tasks) };
  },

  getTask: async (id: number): Promise<Task> => task(await Svc.GetTask(id)),
  subtasks: async (id: number): Promise<Task[]> => tasks(await Svc.Subtasks(id)),
  addTask: async (input: NewTask): Promise<Task> => task(await Svc.AddTask(input)),
  updateTask: async (id: number, patch: Partial<TaskPatch>): Promise<Task> =>
    task(await Svc.UpdateTask(id, { ...emptyPatch, ...patch })),
  complete: (id: number): Promise<void> => Svc.Complete(id),
  reopen: (id: number): Promise<void> => Svc.Reopen(id),
  cancel: (id: number): Promise<void> => Svc.Cancel(id),
  deleteTask: (id: number): Promise<void> => Svc.DeleteTask(id),
  placeTask: (id: number, p: Partial<Place>): Promise<void> => Svc.PlaceTask(id, { ...emptyPlace, ...p }),
  tags: async (): Promise<TagDTO[]> => (await Svc.Tags()) ?? [],

  setRepeat: async (id: number, r: Omit<RepeatDTO, "text">): Promise<Task> => task(await Svc.SetRepeat(id, { ...r, text: "" })),
  removeRepeat: async (id: number): Promise<Task> => task(await Svc.RemoveRepeat(id)),

  addArea: async (name: string): Promise<Area> => area(await Svc.AddArea(name)),
  renameArea: (id: number, name: string): Promise<void> => Svc.RenameArea(id, name),
  deleteArea: (id: number): Promise<void> => Svc.DeleteArea(id),
  addProject: (name: string, areaId: number | null): Promise<Project> => Svc.AddProject(name, areaId),
  renameProject: (id: number, name: string): Promise<void> => Svc.RenameProject(id, name),
  moveProject: (id: number, areaId: number | null): Promise<void> => Svc.MoveProject(id, areaId),
  placeProject: (id: number, change: { areaId?: number | null; afterId: number | null; beforeId: number | null }): Promise<void> =>
    Svc.PlaceProject(id, change.areaId !== undefined, change.areaId ?? null, change.afterId, change.beforeId),
  reorderArea: (id: number, afterId: number | null, beforeId: number | null): Promise<void> => Svc.ReorderArea(id, afterId, beforeId),
  reorderHeading: (id: number, afterId: number | null, beforeId: number | null): Promise<void> => Svc.ReorderHeading(id, afterId, beforeId),
  completeProject: (id: number): Promise<void> => Svc.CompleteProject(id),
  cancelProject: (id: number): Promise<void> => Svc.CancelProject(id),
  reopenProject: (id: number): Promise<void> => Svc.ReopenProject(id),
  deleteProject: (id: number): Promise<void> => Svc.DeleteProject(id),
  addHeading: (projectId: number, title: string): Promise<HeadingDTO> => Svc.AddHeading(projectId, title),
  renameHeading: (id: number, title: string): Promise<void> => Svc.RenameHeading(id, title),
  deleteHeading: (id: number): Promise<void> => Svc.DeleteHeading(id),

  addReminder: (taskId: number | null, title: string, atIso: string): Promise<ReminderDTO> => Svc.AddReminder(taskId, title, atIso),
  pendingReminders: async (): Promise<ReminderDTO[]> => (await Svc.PendingReminders()) ?? [],

  habits: async (): Promise<Habit[]> => ((await Svc.Habits()) ?? []).map((h) => ({ ...h, days: h.days ?? [] })),
  addHabit: async (name: string): Promise<Habit> => ({ ...(await Svc.AddHabit(name)), days: [] }),
  renameHabit: (id: number, name: string): Promise<void> => Svc.RenameHabit(id, name),
  deleteHabit: (id: number): Promise<void> => Svc.DeleteHabit(id),
  setHabitDay: (id: number, day: string, done: boolean): Promise<void> => Svc.SetHabitDay(id, day, done),

  getPrefs: (): Promise<Prefs> => Svc.GetPrefs(),
  setPrefs: (p: Prefs): Promise<Prefs> => Svc.SetPrefs(p),
};

const emptyPatch: TaskPatch = {
  title: null, notes: null, whenKind: null, whenDate: null, deadline: null, evening: null, tags: null,
  projectId: null, clearProject: false, areaId: null, headingId: null, clearHeading: false,
};

const emptyPlace: Place = { afterId: null, beforeId: null, setHeading: false, headingId: null, setEvening: false, evening: false };

/** Subscribe to core change events (task edits, list invalidations, external writes). Returns an unsubscribe. */
export function onChange(handler: (e: ChangeEvent) => void): () => void {
  return Events.On("zenith:change", (ev: { data: ChangeEvent }) => handler(ev.data));
}
