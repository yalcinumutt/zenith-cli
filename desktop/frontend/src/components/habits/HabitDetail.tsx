import { ArrowLeftIcon, PencilIcon, Trash2Icon } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { Habit } from "../../api/client";
import { deleteHabit, renameHabit, toggleHabitDay } from "../../commands/actions";
import { bestStreak, daySet, rate, streak } from "../../lib/habits";
import { Heatmap } from "./Heatmap";

/** One habit's history: stats and a 26-week heatmap you can edit by clicking days. */
export function HabitDetail({ habit, today, onBack }: { habit: Habit; today: string; onBack: () => void }) {
  const days = daySet(habit);
  return (
    <div>
      <Button variant="ghost" size="sm" className="-ml-2 mb-3 text-muted-foreground" onClick={onBack}><ArrowLeftIcon data-icon="inline-start" />Habits</Button>
      <div className="mb-6 flex items-start justify-between gap-4">
        <h1 className="text-3xl font-semibold tracking-tight">{habit.name}</h1>
        <div className="flex shrink-0 gap-1 pt-2">
          <Button variant="ghost" size="icon-sm" aria-label="Rename habit" onClick={() => renameHabit(habit)}><PencilIcon /></Button>
          <Button variant="ghost" size="icon-sm" aria-label="Delete habit" onClick={() => deleteHabit(habit, onBack)}><Trash2Icon /></Button>
        </div>
      </div>
      <div className="mb-6 grid grid-cols-4 gap-3">
        <Stat label="Current streak" value={`${streak(days, today)} d`} />
        <Stat label="Best streak" value={`${bestStreak(days)} d`} />
        <Stat label="Last 30 days" value={`${Math.round(rate(days, today, 30) * 100)}%`} />
        <Stat label="Check-ins" value={String(days.size)} hint="past 13 months" />
      </div>
      <h2 className="mb-2 text-sm font-semibold">Last 26 weeks</h2>
      <Heatmap name={habit.name} days={days} today={today} onToggle={(d) => void toggleHabitDay(habit, d)} />
      <p className="mt-3 text-xs text-muted-foreground">Click a day to tick or un-tick it.</p>
    </div>
  );
}

export function Stat({ label, value, hint }: { label: string; value: string; hint?: string }) {
  return (
    <div className="rounded-xl border bg-card px-4 py-3">
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className="text-2xl font-semibold tabular-nums">{value}</div>
      {hint && <div className="truncate text-xs text-muted-foreground">{hint}</div>}
    </div>
  );
}
