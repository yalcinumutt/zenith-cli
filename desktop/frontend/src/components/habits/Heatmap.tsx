import { addDays } from "@/lib/dates";
import { cn } from "@/lib/utils";
import { weekStart, type DaySet } from "../../lib/habits";

/** A weeks x 7 grid of days, Monday first, newest week last. Click a day to tick or un-tick it. */
export function Heatmap({ name, days, today, weeks = 26, onToggle }: { name: string; days: DaySet; today: string; weeks?: number; onToggle: (day: string) => void }) {
  const start = addDays(weekStart(today), -(weeks - 1) * 7);
  return (
    <div className="grid grid-flow-col grid-rows-7 gap-[3px]" style={{ gridTemplateColumns: `repeat(${weeks}, minmax(0, 1fr))` }}>
      {Array.from({ length: weeks * 7 }, (_, i) => {
        const day = addDays(start, i);
        if (day > today) return <span key={i} className="aspect-square" />;
        const done = days.has(day);
        return (
          <button
            key={i}
            title={day}
            aria-label={`${name} on ${day}: ${done ? "done" : "not done"}`}
            aria-pressed={done}
            onClick={() => onToggle(day)}
            className={cn("aspect-square rounded-[3px]", done ? "bg-primary" : "bg-muted hover:bg-border", day === today && "ring-1 ring-foreground")}
          />
        );
      })}
    </div>
  );
}
