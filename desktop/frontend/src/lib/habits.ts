// Pure habit maths over the days a habit was done (YYYY-MM-DD strings).
import type { Habit } from "../api/client";
import { addDays, daysBetween } from "./dates";

export type DaySet = ReadonlySet<string>;

export const daySet = (h: Habit): DaySet => new Set(h.days);

/** Consecutive done days ending today, or yesterday while today is still open. */
export function streak(days: DaySet, today: string): number {
  let d = days.has(today) ? today : addDays(today, -1);
  let n = 0;
  while (days.has(d)) { n++; d = addDays(d, -1); }
  return n;
}

/** The longest run of consecutive done days in the whole history. */
export function bestStreak(days: DaySet): number {
  const sorted = [...days].sort();
  let best = 0;
  let run = 0;
  sorted.forEach((d, i) => {
    run = i > 0 && daysBetween(sorted[i - 1], d) === 1 ? run + 1 : 1;
    best = Math.max(best, run);
  });
  return best;
}

export function lastDays(today: string, n: number): string[] {
  return Array.from({ length: n }, (_, i) => addDays(today, i - (n - 1)));
}

/** Share of the last n days (today included) the habit was done, 0..1. */
export function rate(days: DaySet, today: string, n: number): number {
  return lastDays(today, n).filter((d) => days.has(d)).length / n;
}

/** Monday of the week containing a date. */
export function weekStart(date: string): string {
  const dow = new Date(`${date}T00:00:00`).getDay(); // 0 = Sunday
  return addDays(date, -((dow + 6) % 7));
}

export const WEEKDAY_LETTERS = ["M", "T", "W", "T", "F", "S", "S"];
export const weekdayLetter = (date: string): string => WEEKDAY_LETTERS[(new Date(`${date}T00:00:00`).getDay() + 6) % 7];
