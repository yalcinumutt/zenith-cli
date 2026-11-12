import { FlameIcon, PlusIcon } from "lucide-react";
import { useState } from "react";
import { ContextMenu, ContextMenuContent, ContextMenuItem, ContextMenuTrigger } from "@/components/ui/context-menu";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { api, type Habit } from "../../api/client";
import { attempt, deleteHabit, renameHabit, toggleHabitDay } from "../../commands/actions";
import { daySet, lastDays, streak, weekdayLetter } from "../../lib/habits";
import { RoundCheck } from "../ui";

/** The daily checklist: tick today, see the last week and the streak at a glance, open a habit for its history. */
export function TodayTab({ habits, today, onOpen }: { habits: Habit[]; today: string; onOpen: (id: number) => void }) {
  const done = habits.filter((h) => h.days.includes(today)).length;
  const week = lastDays(today, 7);
  return (
    <div>
      {habits.length > 0 && (
        <div className="mb-5">
          <p className="text-sm text-muted-foreground">{done} of {habits.length} done today</p>
          <div className="mt-2 h-1.5 overflow-hidden rounded-full bg-muted" role="progressbar" aria-valuemin={0} aria-valuemax={habits.length} aria-valuenow={done} aria-label="Habits done today">
            <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${(done / habits.length) * 100}%` }} />
          </div>
        </div>
      )}
      <ul className="flex flex-col">
        {habits.map((h) => {
          const days = daySet(h);
          const d = days.has(today);
          const s = streak(days, today);
          return (
            <li key={h.id}>
              <ContextMenu>
                <ContextMenuTrigger asChild>
                  <div className="flex items-center gap-3 rounded-xl px-3 py-2.5 hover:bg-muted">
                    <RoundCheck checked={d} label={`${d ? "Un-log" : "Log"} ${h.name} today`} onToggle={() => void toggleHabitDay(h, today)} />
                    <button onClick={() => onOpen(h.id)} className={cn("flex-1 text-left text-[15px]", d && "text-muted-foreground line-through")}>{h.name}</button>
                    <span className="flex items-center gap-1" role="img" aria-label={`Done on ${week.filter((x) => days.has(x)).length} of the last 7 days`}>
                      {week.map((day) => (
                        <span key={day} title={day} className="flex flex-col items-center gap-0.5">
                          <span className={cn("size-2.5 rounded-full", days.has(day) ? "bg-primary" : "bg-border")} />
                          <span className="text-[9px] text-muted-foreground" aria-hidden>{weekdayLetter(day)}</span>
                        </span>
                      ))}
                    </span>
                    <span className={cn("flex w-12 items-center justify-end gap-1 text-sm tabular-nums", s > 0 ? "text-amber-600 dark:text-amber-400" : "text-muted-foreground")}>
                      <FlameIcon className="size-3.5" aria-hidden /><span className="sr-only">Streak </span>{s}
                    </span>
                  </div>
                </ContextMenuTrigger>
                <ContextMenuContent className="w-44">
                  <ContextMenuItem onSelect={() => onOpen(h.id)}>History</ContextMenuItem>
                  <ContextMenuItem onSelect={() => setTimeout(() => renameHabit(h), 0)}>Rename…</ContextMenuItem>
                  <ContextMenuItem variant="destructive" onSelect={() => setTimeout(() => deleteHabit(h), 0)}>Delete…</ContextMenuItem>
                </ContextMenuContent>
              </ContextMenu>
            </li>
          );
        })}
      </ul>
      {habits.length === 0 && <p className="py-4 text-sm text-muted-foreground">No habits yet. Add one below.</p>}
      <NewHabit />
    </div>
  );
}

function NewHabit() {
  const [text, setText] = useState("");
  return (
    <div className="mt-2 flex items-center gap-3 rounded-lg px-3 py-2 focus-within:bg-muted">
      <PlusIcon className="size-4 text-muted-foreground" aria-hidden />
      <Input
        aria-label="New habit"
        placeholder="New habit"
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => {
          if (e.key === "Escape") (e.target as HTMLInputElement).blur();
          if (e.key === "Enter" && text.trim()) {
            const name = text;
            setText("");
            void attempt(() => api.addHabit(name));
          }
        }}
        className="h-8 border-transparent bg-transparent px-0 text-[15px] shadow-none focus-visible:ring-0 dark:bg-transparent"
      />
    </div>
  );
}
