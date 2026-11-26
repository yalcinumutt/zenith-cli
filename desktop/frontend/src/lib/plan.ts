// Time-block helpers. Times are minutes after local midnight; days are YYYY-MM-DD.
import type { Block } from "../api/client";

export const DAY_MIN = 24 * 60;
export const SNAP = 15;
export const HOUR_PX = 64;
export const PX_PER_MIN = HOUR_PX / 60;
export const DEFAULT_BLOCK = 30;
export const ESTIMATES = [15, 30, 45, 60, 90, 120, 180, 240];

export const snap = (m: number): number => Math.round(m / SNAP) * SNAP;
export const clamp = (n: number, lo: number, hi: number): number => Math.min(hi, Math.max(lo, n));

/** 570 -> "09:30". */
export function fmtClock(m: number): string {
  const h = Math.floor(m / 60) % 24;
  return `${String(h).padStart(2, "0")}:${String(m % 60).padStart(2, "0")}`;
}

/** 90 -> "1h 30m", 45 -> "45m", 120 -> "2h". */
export function fmtDuration(m: number): string {
  const h = Math.floor(m / 60);
  const r = m % 60;
  if (h === 0) return `${r}m`;
  return r === 0 ? `${h}h` : `${h}h ${r}m`;
}

/** The earliest start at or after `from` where `duration` fits between the blocks, or null. */
export function firstFreeSlot(blocks: Block[], duration: number, from: number): number | null {
  const sorted = [...blocks].sort((a, b) => a.start - b.start);
  let t = clamp(Math.ceil(from / SNAP) * SNAP, 0, DAY_MIN);
  for (const b of sorted) {
    if (b.start + b.duration <= t) continue;
    if (t + duration <= b.start) break;
    t = Math.ceil((b.start + b.duration) / SNAP) * SNAP;
  }
  return t + duration <= DAY_MIN ? t : null;
}

/** Minutes of a task already placed in blocks. */
export function scheduledMinutes(blocks: Block[], taskId: number): number {
  return blocks.filter((b) => b.taskId === taskId).reduce((n, b) => n + b.duration, 0);
}

/** 570 -> "09:30" for <input type="time">; and back. */
export const toTimeInput = (m: number): string => fmtClock(m);
export function fromTimeInput(v: string): number | null {
  const m = /^(\d{1,2}):(\d{2})$/.exec(v);
  return m ? Number(m[1]) * 60 + Number(m[2]) : null;
}
