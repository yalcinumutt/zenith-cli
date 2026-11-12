import { BellIcon, Trash2Icon, XIcon } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { api, type Repeat, type Task } from "../api/client";
import { attempt, toggleComplete } from "../commands/actions";
import { addDays } from "../lib/dates";
import { useInfo, useSubtasks, useTags } from "../state/queries";
import { useUI } from "../state/ui";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { ToggleGroup, ToggleGroupItem } from "@/components/ui/toggle-group";
import { DatePicker } from "./DatePicker";
import { RoundCheck } from "./ui";

const WEEKDAYS = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];

/** The title while a task is open: an input in place of the row's text. */
export function TitleEditor({ task }: { task: Task }) {
  const [title, setTitle] = useState(task.title);
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => setTitle(task.title), [task.title]);
  useEffect(() => { ref.current?.focus(); }, [task.id]);
  const commit = () => {
    if (title.trim() === task.title) return;
    if (!title.trim()) { setTitle(task.title); return; }
    void attempt(() => api.updateTask(task.id, { title }));
  };
  return (
    <Input
      ref={ref}
      aria-label="Title"
      value={title}
      onClick={(e) => e.stopPropagation()}
      onDoubleClick={(e) => e.stopPropagation()}
      onChange={(e) => setTitle(e.target.value)}
      onBlur={commit}
      onKeyDown={(e) => { if (e.key === "Enter") { commit(); (e.target as HTMLInputElement).blur(); } }}
      className="h-7 border-transparent bg-transparent px-1 text-[15px] leading-snug shadow-none dark:bg-transparent"
    />
  );
}

/** The expanded, in-place card: notes, checklist of subtasks, and the When / Deadline / repeat footer. */
export function TaskCard({ task }: { task: Task }) {
  const today = useInfo().data?.today ?? "";
  const subtasks = useSubtasks(task.id, true).data ?? [];
  const [notes, setNotes] = useState(task.notes);
  const [newSub, setNewSub] = useState("");

  // Keep local text in step when the task changes underneath (CLI edit, other field).
  useEffect(() => setNotes(task.notes), [task.notes]);

  const save = (patch: Parameters<typeof api.updateTask>[1]) => void attempt(() => api.updateTask(task.id, patch));

  return (
    <div className="flex flex-col gap-3 px-3 pb-3 pl-[2.9rem] pr-4" onClick={(e) => e.stopPropagation()} onDoubleClick={(e) => e.stopPropagation()}>
      <Textarea
        aria-label="Notes"
        placeholder="Notes"
        value={notes}
        rows={notes.split("\n").length > 2 ? 4 : 2}
        onChange={(e) => setNotes(e.target.value)}
        onBlur={() => notes !== task.notes && save({ notes })}
        className="min-h-0 resize-none border-transparent bg-transparent px-1 shadow-none dark:bg-transparent"
      />

      <div className="flex flex-col gap-1.5" role="group" aria-label="Checklist">
        {subtasks.map((s) => (
          <div key={s.id} className="flex items-center gap-2.5">
            <RoundCheck small checked={s.status !== "open"} label={`${s.status === "open" ? "Complete" : "Reopen"} subtask ${s.title}`} onToggle={() => void toggleComplete(s)} />
            <span className={s.status !== "open" ? "text-sm text-muted-foreground line-through" : "text-sm"}>{s.title}</span>
          </div>
        ))}
        <Input
          aria-label="Add a subtask"
          placeholder="Add subtask"
          value={newSub}
          onChange={(e) => setNewSub(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && newSub.trim()) {
              const text = newSub;
              setNewSub("");
              void attempt(() => api.addTask({ title: text, notes: "", whenKind: "", whenDate: "", deadline: "", evening: false, projectId: null, areaId: null, headingId: null, parentId: task.id, tags: [] }));
            }
          }}
          className="h-7 border-transparent bg-transparent px-1 shadow-none dark:bg-transparent"
        />
      </div>

      <TagsField task={task} />

      <Separator />
      <div className="flex flex-wrap items-center gap-x-5 gap-y-2 text-xs text-muted-foreground">
        <WhenField task={task} today={today} save={save} />
        <span className="flex items-center gap-1.5">Deadline
          <DatePicker label="Deadline" placeholder="None" today={today} value={task.deadline} onChange={(d) => save({ deadline: d })} clearable />
        </span>
        <RepeatField task={task} />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2">
        <ReminderField task={task} />
        <div className="flex gap-2">
          {task.status === "open" && <Button variant="ghost" size="sm" onClick={() => void attempt(() => api.cancel(task.id))}>Cancel task</Button>}
          <Button variant="ghost" size="icon-sm" aria-label="Delete task" onClick={() => {
            useUI.getState().openDialog({ kind: "confirm", title: "Delete task", message: `Delete “${task.title}” and its subtasks?`, confirm: "Delete", danger: true, onConfirm: () => { useUI.getState().expand(null); void attempt(() => api.deleteTask(task.id)); } });
          }}><Trash2Icon /></Button>
        </div>
      </div>
    </div>
  );
}

function TagsField({ task }: { task: Task }) {
  const all = useTags().data ?? [];
  const [text, setText] = useState("");
  const set = (names: string[]) => void attempt(() => api.updateTask(task.id, { tags: names }));
  const add = () => {
    const names = text.split(",").map((s) => s.trim()).filter(Boolean);
    if (names.length) set([...task.tags.map((t) => t.name), ...names]);
    setText("");
  };
  const listId = `tags-${task.id}`;
  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {task.tags.map((t) => (
        <Badge key={t.id} variant="secondary" className="gap-1">
          #{t.name}
          <button aria-label={`Remove tag ${t.name}`} className="rounded hover:text-destructive" onClick={() => set(task.tags.filter((x) => x.id !== t.id).map((x) => x.name))}><XIcon /></button>
        </Badge>
      ))}
      <Input aria-label="Add tags" list={listId} placeholder="Add tags" value={text} onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => { if (e.key === "Enter" || e.key === ",") { e.preventDefault(); add(); } }} onBlur={add}
        className="h-6 w-28 border-transparent bg-transparent px-1 text-xs shadow-none dark:bg-transparent" />
      <datalist id={listId}>{all.map((t) => <option key={t.id} value={t.name} />)}</datalist>
    </div>
  );
}

type SaveFn = (patch: Parameters<typeof api.updateTask>[1]) => void;

function whenValue(task: Task, today: string): string {
  if (task.whenKind === "someday") return "someday";
  if (task.whenKind !== "scheduled") return "anytime";
  if (task.whenDate === today) return "today";
  if (task.whenDate === addDays(today, 1)) return "tomorrow";
  return "date";
}

function WhenField({ task, today, save }: { task: Task; today: string; save: SaveFn }) {
  const value = whenValue(task, today);
  const choose = (v: string) => {
    if (v === "anytime") save({ whenKind: "anytime", whenDate: "" });
    else if (v === "someday") save({ whenKind: "someday", whenDate: "" });
    else if (v === "today") save({ whenKind: "scheduled", whenDate: today });
    else if (v === "tomorrow") save({ whenKind: "scheduled", whenDate: addDays(today, 1) });
    else save({ whenKind: "scheduled", whenDate: task.whenDate || addDays(today, 2) });
  };
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      When
      <Select value={value} onValueChange={choose}>
        <SelectTrigger size="sm" aria-label="When" className="w-36"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value="anytime">Anytime</SelectItem>
            <SelectItem value="today">Today</SelectItem>
            <SelectItem value="tomorrow">Tomorrow</SelectItem>
            <SelectItem value="date">Pick a date…</SelectItem>
            <SelectItem value="someday">Someday</SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
      {value === "date" && (
        <DatePicker label="Scheduled date" today={today} value={task.whenDate} onChange={(d) => d && save({ whenKind: "scheduled", whenDate: d })} />
      )}
      {task.whenKind === "scheduled" && (
        <span className="flex items-center gap-1.5">
          <Switch id={`evening-${task.id}`} size="sm" checked={task.evening} onCheckedChange={(c) => save({ evening: c })} />
          <Label htmlFor={`evening-${task.id}`} className="text-xs font-normal">This evening</Label>
        </span>
      )}
    </span>
  );
}

function RepeatField({ task }: { task: Task }) {
  const r = task.repeat;
  const [kind, setKind] = useState<string>(r?.kind ?? "none");
  const [interval, setInterval_] = useState(r?.interval ?? 1);
  const [days, setDays] = useState<number[]>(r?.weekdays ?? []);
  const [monthDay, setMonthDay] = useState(r?.monthDay ?? 0);
  useEffect(() => {
    setKind(r?.kind ?? "none"); setInterval_(r?.interval ?? 1); setDays(r?.weekdays ?? []); setMonthDay(r?.monthDay ?? 0);
  }, [r?.kind, r?.interval, r?.monthDay, r?.weekdays]);

  const apply = (next: { kind: string; interval: number; days: number[]; monthDay: number }) => {
    if (next.kind === "none") { void attempt(() => api.removeRepeat(task.id)); return; }
    const rule: Omit<Repeat, "text"> = { kind: next.kind, interval: Math.max(1, next.interval || 1), weekdays: next.kind === "weeks" ? next.days : [], monthDay: next.kind === "months" ? next.monthDay : 0 };
    void attempt(() => api.setRepeat(task.id, rule));
  };
  const change = (patch: Partial<{ kind: string; interval: number; days: number[]; monthDay: number }>) => {
    const next = { kind, interval, days, monthDay, ...patch };
    setKind(next.kind); setInterval_(next.interval); setDays(next.days); setMonthDay(next.monthDay);
    apply(next);
  };
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      Repeat
      <Select value={kind} onValueChange={(v) => change({ kind: v })}>
        <SelectTrigger size="sm" aria-label="Repeat" className="w-36"><SelectValue /></SelectTrigger>
        <SelectContent>
          <SelectGroup>
            <SelectItem value="none">Never</SelectItem>
            <SelectItem value="days">Every N days</SelectItem>
            <SelectItem value="weeks">Every N weeks</SelectItem>
            <SelectItem value="months">Every N months</SelectItem>
          </SelectGroup>
        </SelectContent>
      </Select>
      {kind !== "none" && (
        <Input type="number" min={1} max={999} aria-label="Repeat interval" value={interval} onChange={(e) => change({ interval: Number(e.target.value) })} className="h-7 w-16" />
      )}
      {kind === "weeks" && (
        <ToggleGroup type="multiple" size="sm" variant="outline" aria-label="Weekdays" value={days.map(String)}
          onValueChange={(v) => change({ days: v.map(Number).sort() })}>
          {WEEKDAYS.map((name, d) => <ToggleGroupItem key={name} value={String(d)} aria-label={name} className="px-1.5 text-[11px]">{name}</ToggleGroupItem>)}
        </ToggleGroup>
      )}
      {kind === "months" && (
        <span className="flex items-center gap-1.5">on day
          <Input type="number" min={1} max={31} aria-label="Day of month" placeholder="same" value={monthDay || ""} onChange={(e) => change({ monthDay: Number(e.target.value) })} className="h-7 w-16" />
        </span>
      )}
      {r && <span className="text-foreground">{r.text}</span>}
    </span>
  );
}

function ReminderField({ task }: { task: Task }) {
  const focus = useUI((s) => s.reminderFocus);
  const ref = useRef<HTMLInputElement>(null);
  const defaultAt = () => {
    const d = new Date(Date.now() + 30 * 60_000);
    d.setSeconds(0, 0);
    const pad = (n: number) => String(n).padStart(2, "0");
    return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  };
  const [at, setAt] = useState(defaultAt);
  useEffect(() => { if (focus > 0) ref.current?.focus(); }, [focus]);
  const add = async () => {
    const when = new Date(at);
    if (Number.isNaN(when.getTime())) return;
    const r = await attempt(() => api.addReminder(task.id, task.title, when.toISOString()));
    if (r) useUI.getState().flash(`Reminder set for ${when.toLocaleString()}`);
  };
  return (
    <span className="flex items-center gap-1.5 text-xs text-muted-foreground">
      <BellIcon className="size-3.5" aria-hidden />
      <Input ref={ref} type="datetime-local" aria-label="Reminder time" value={at} onChange={(e) => setAt(e.target.value)} className="h-7 w-auto"
        onKeyDown={(e) => { if (e.key === "Enter") void add(); }} />
      <Button size="sm" variant="outline" onClick={() => void add()}>Remind me</Button>
    </span>
  );
}
