import { ChevronLeftIcon, ChevronRightIcon } from "lucide-react";
import { useState } from "react";
import { Button } from "@/components/ui/button";
import { addDays, fmtISO, parseDay } from "@/lib/dates";
import { cn } from "@/lib/utils";
import type { Habit } from "../../api/client";
import { toggleHabitDay } from "../../commands/actions";
import { WEEKDAY_LETTERS, daySet, lastDays, rate, streak } from "../../lib/habits";
import { Stat } from "./HabitDetail";

/** The month calendar (shaded by how much of the day was done), a day editor, and 30-day bars per habit. */
export function InsightsTab({ habits, today, onOpen }: { habits: Habit[]; today: string; onOpen: (id: number) => void }) {
  const [month, setMonth] = useState(today.slice(0, 7));
  const [sel, setSel] = useState(today);
  if (habits.length === 0) return <p className="py-4 text-sm text-muted-foreground">Add a habit to see insights.</p>;

  const sets = new Map(habits.map((h) => [h.id, daySet(h)]));
  const first = `${month}-01`;
  const lead = (parseDay(first).getDay() + 6) % 7; // Monday first
  const daysInMonth = new Date(Number(month.slice(0, 4)), Number(month.slice(5, 7)), 0).getDate();
  const cells = Array.from({ length: lead + daysInMonth }, (_, i) => (i < lead ? null : addDays(first, i - lead)));
  const shift = (n: number) => { const d = parseDay(first); d.setMonth(d.getMonth() + n); setMonth(fmtISO(d).slice(0, 7)); };
  const frac = (d: string) => habits.filter((h) => sets.get(h.id)!.has(d)).length / habits.length;
  const best = [...habits].sort((a, b) => streak(sets.get(b.id)!, today) - streak(sets.get(a.id)!, today))[0];
  const overall = habits.reduce((n, h) => n + rate(sets.get(h.id)!, today, 30), 0) / habits.length;
  const doneToday = habits.filter((h) => sets.get(h.id)!.has(today)).length;

  return (
    <div>
      <div className="mb-6 grid grid-cols-3 gap-3">
        <Stat label="Today" value={`${doneToday}/${habits.length}`} />
        <Stat label="30-day rate" value={`${Math.round(overall * 100)}%`} />
        <Stat label="Best streak" value={`${streak(sets.get(best.id)!, today)} d`} hint={best.name} />
      </div>
      <div className="grid grid-cols-[1fr_15rem] gap-6">
        <div>
          <div className="mb-2 flex items-center justify-between">
            <h2 className="text-sm font-semibold">{parseDay(first).toLocaleDateString(undefined, { month: "long", year: "numeric" })}</h2>
            <span className="flex gap-1">
              <Button variant="ghost" size="icon-xs" aria-label="Previous month" onClick={() => shift(-1)}><ChevronLeftIcon /></Button>
              <Button variant="ghost" size="icon-xs" aria-label="Next month" onClick={() => shift(1)} disabled={month >= today.slice(0, 7)}><ChevronRightIcon /></Button>
            </span>
          </div>
          <div className="grid grid-cols-7 gap-1">
            {WEEKDAY_LETTERS.map((l, i) => <div key={i} className="text-center text-[11px] text-muted-foreground">{l}</div>)}
            {cells.map((d, i) => d === null ? <span key={i} /> : (
              <button
                key={d}
                onClick={() => setSel(d)}
                disabled={d > today}
                aria-label={`${d}: ${Math.round(frac(d) * 100)}% done`}
                aria-pressed={d === sel}
                className={cn("flex aspect-square items-center justify-center rounded-lg text-xs", d > today && "opacity-30", d === sel && "ring-2 ring-ring", d === today && "font-semibold")}
                style={d <= today ? { background: `color-mix(in oklab, var(--primary) ${Math.round(frac(d) * 70)}%, var(--muted))`, color: frac(d) > 0.6 ? "var(--primary-foreground)" : undefined } : undefined}
              >
                {Number(d.slice(8))}
              </button>
            ))}
          </div>
        </div>
        <div>
          <h2 className="mb-2 text-sm font-semibold">{sel === today ? "Today" : sel}</h2>
          <ul className="flex flex-col gap-1">
            {habits.map((h) => {
              const on = sets.get(h.id)!.has(sel);
              return (
                <li key={h.id}>
                  <button onClick={() => void toggleHabitDay(h, sel)} aria-pressed={on} className="flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-sm hover:bg-muted">
                    <span className={cn("size-3 shrink-0 rounded-full border", on ? "border-primary bg-primary" : "border-muted-foreground")} />
                    <span className={cn("truncate", on && "text-muted-foreground")}>{h.name}</span>
                  </button>
                </li>
              );
            })}
          </ul>
          <h2 className="mb-2 mt-6 text-sm font-semibold">Last 30 days</h2>
          <ul className="flex flex-col gap-1">
            {habits.map((h) => (
              <li key={h.id}>
                <button onClick={() => onOpen(h.id)} className="flex w-full items-center gap-2 rounded-md py-1 hover:bg-muted" aria-label={`${h.name}: open history`}>
                  <span className="w-24 truncate text-left text-xs text-muted-foreground">{h.name}</span>
                  <span className="flex h-4 flex-1 items-end gap-px">
                    {lastDays(today, 30).map((d) => <span key={d} className={cn("flex-1 rounded-[1px]", sets.get(h.id)!.has(d) ? "h-4 bg-primary" : "h-1 bg-border")} />)}
                  </span>
                </button>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </div>
  );
}
