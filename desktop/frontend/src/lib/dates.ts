// Dates are plain YYYY-MM-DD strings end to end; "today" comes from the Go side
// (the core's clock), never from the web view, so every front-end agrees.

const parse = (d: string): Date => new Date(`${d}T00:00:00`);

export function addDays(date: string, n: number): string {
  const d = parse(date);
  d.setDate(d.getDate() + n);
  return fmtISO(d);
}

export function fmtISO(d: Date): string {
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${d.getFullYear()}-${m}-${day}`;
}

export function daysBetween(from: string, to: string): number {
  return Math.round((parse(to).getTime() - parse(from).getTime()) / 86_400_000);
}

const weekday = (d: string) => parse(d).toLocaleDateString(undefined, { weekday: "long" });
const short = (d: string) => parse(d).toLocaleDateString(undefined, { day: "numeric", month: "short" });

/** "Today", "Tomorrow", a weekday within the next week, otherwise "12 Jan". */
export function friendlyDay(date: string, today: string): string {
  const n = daysBetween(today, date);
  if (n === 0) return "Today";
  if (n === 1) return "Tomorrow";
  if (n === -1) return "Yesterday";
  if (n > 1 && n < 7) return weekday(date);
  return short(date);
}

/** Title for an Upcoming day group: "Tomorrow", or "Friday, 14 Nov". */
export function upcomingTitle(date: string, today: string): string {
  const n = daysBetween(today, date);
  if (n === 1) return "Tomorrow";
  return `${weekday(date)}, ${short(date)}`;
}

export function deadlineLabel(deadline: string, today: string): string {
  const n = daysBetween(deadline, today);
  if (n > 0) return `${n}d overdue`;
  return friendlyDay(deadline, today);
}

export function formatSeconds(s: number): string {
  if (s < 60) return `${s}s`;
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}
