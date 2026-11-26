import { useEffect, useRef, useState } from "react";
import { useDroppable } from "@dnd-kit/core";
import { useQueryClient } from "@tanstack/react-query";
import { XIcon } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { api, type Block, type Busy, type Proposal } from "../../api/client";
import { clamp, DAY_MIN, fmtClock, fmtDuration, HOUR_PX, PX_PER_MIN, SNAP, snap } from "../../lib/plan";
import { keys } from "../../state/queries";

export const TIMELINE_DROP_ID = "timeline";
const GUTTER = 52; // px reserved for the hour labels
const FIRST_VISIBLE_HOUR = 7;

interface Props {
  date: string;
  blocks: Block[];
  /** Calendar events: locked, drawn behind the blocks. */
  busy: Busy[];
  /** Suggested blocks awaiting review, drawn as dashed ghosts. */
  proposals: Proposal[];
  workStart: number;
  workEnd: number;
  /** Minute of "now" when the day shown is today, else null. */
  nowMin: number | null;
}

/** A day as a vertical ruler. Blocks are dragged to move and pulled from the bottom edge to resize. */
export function DayTimeline({ date, blocks, busy, proposals, workStart, workEnd, nowMin }: Props) {
  const scroller = useRef<HTMLDivElement>(null);
  const { setNodeRef } = useDroppable({ id: TIMELINE_DROP_ID });

  useEffect(() => {
    if (scroller.current) scroller.current.scrollTop = FIRST_VISIBLE_HOUR * HOUR_PX - 8;
  }, [date]);

  return (
    <div ref={scroller} className="h-full overflow-y-auto rounded-xl border bg-card" aria-label="Day timeline">
      <div ref={setNodeRef} className="relative" style={{ height: DAY_MIN * PX_PER_MIN }} data-timeline>
        {Array.from({ length: 24 }, (_, h) => (
          <div key={h} className="absolute left-0 right-0 border-t border-border/60" style={{ top: h * HOUR_PX }}>
            <span className="absolute -top-2 left-2 bg-card px-0.5 text-[11px] tabular-nums text-muted-foreground">{fmtClock(h * 60)}</span>
          </div>
        ))}
        <div className="pointer-events-none absolute inset-x-0 top-0 bg-muted/40" style={{ height: workStart * PX_PER_MIN }} aria-hidden />
        <div className="pointer-events-none absolute inset-x-0 bg-muted/40" style={{ top: workEnd * PX_PER_MIN, height: (DAY_MIN - workEnd) * PX_PER_MIN }} aria-hidden />
        {busy.map((b, i) => (
          <div
            key={`busy${i}`}
            title={`${b.title || "Busy"} · ${fmtClock(b.start)}–${fmtClock(b.end)}`}
            aria-label={`Calendar: ${b.title || "Busy"}, ${fmtClock(b.start)} to ${fmtClock(b.end)}`}
            className="pointer-events-none absolute z-0 overflow-hidden rounded-lg border border-border bg-[repeating-linear-gradient(135deg,transparent_0_6px,var(--border)_6px_7px)] px-2 pt-1 text-xs text-muted-foreground"
            style={{ top: b.start * PX_PER_MIN + 1, height: (b.end - b.start) * PX_PER_MIN - 2, left: GUTTER, right: 8 }}
          >
            <span className="truncate font-medium">{b.title || "Busy"}</span>
          </div>
        ))}
        {proposals.map((p) => (
          <div
            key={`ghost${p.taskId}-${p.start}`}
            aria-label={`Suggested: ${p.taskTitle}, ${fmtClock(p.start)} to ${fmtClock(p.start + p.duration)}`}
            className="pointer-events-none absolute z-10 overflow-hidden rounded-lg border-2 border-dashed border-primary/60 bg-primary/5 px-2 pt-1 text-xs"
            style={{ top: p.start * PX_PER_MIN + 1, height: p.duration * PX_PER_MIN - 2, left: GUTTER, right: 8 }}
          >
            <p className="truncate font-medium text-primary">{p.taskTitle}</p>
            {p.duration > 30 && <p className="truncate text-muted-foreground">{fmtClock(p.start)}–{fmtClock(p.start + p.duration)}{p.reason ? ` · ${p.reason}` : ""}</p>}
          </div>
        ))}
        {nowMin !== null && (
          <div className="pointer-events-none absolute left-0 right-0 z-20 border-t-2 border-destructive/70" style={{ top: nowMin * PX_PER_MIN }}>
            <span className="absolute -top-1 left-[46px] size-2 rounded-full bg-destructive" />
          </div>
        )}
        {blocks.map((b) => <BlockCard key={b.id} block={b} date={date} others={blocks} />)}
      </div>
    </div>
  );
}

interface Preview { start: number; duration: number }

function BlockCard({ block, date, others }: { block: Block; date: string; others: Block[] }) {
  const qc = useQueryClient();
  const [preview, setPreview] = useState<Preview | null>(null);
  const shown = preview ?? { start: block.start, duration: block.duration };

  /** Pointer drag: `move` shifts the start, `resize` stretches the end. Commits on release. */
  const begin = (mode: "move" | "resize") => (e: React.PointerEvent<HTMLElement>) => {
    if (e.button !== 0) return;
    e.preventDefault();
    e.stopPropagation();
    const el = e.currentTarget;
    el.setPointerCapture(e.pointerId);
    const y0 = e.clientY;
    let last: Preview = { start: block.start, duration: block.duration };
    let moved = false;

    const compute = (clientY: number): Preview => {
      const delta = snap((clientY - y0) / PX_PER_MIN);
      if (mode === "move") return { start: clamp(block.start + delta, 0, DAY_MIN - block.duration), duration: block.duration };
      return { start: block.start, duration: clamp(block.duration + delta, SNAP, DAY_MIN - block.start) };
    };
    const onMove = (ev: PointerEvent) => {
      last = compute(ev.clientY);
      moved = last.start !== block.start || last.duration !== block.duration;
      setPreview(last);
    };
    const finish = async (ev: PointerEvent) => {
      el.removeEventListener("pointermove", onMove);
      el.removeEventListener("pointerup", finish);
      el.removeEventListener("pointercancel", cancel);
      if (el.hasPointerCapture(ev.pointerId)) el.releasePointerCapture(ev.pointerId);
      if (!moved) { setPreview(null); return; }
      try {
        if (mode === "move") await api.moveBlock(block.id, date, last.start);
        else await api.resizeBlock(block.id, last.duration);
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Could not change that block.");
      }
      await qc.invalidateQueries({ queryKey: keys.plan(date) });
      setPreview(null);
    };
    const cancel = () => {
      el.removeEventListener("pointermove", onMove);
      el.removeEventListener("pointerup", finish);
      el.removeEventListener("pointercancel", cancel);
      setPreview(null);
    };
    el.addEventListener("pointermove", onMove);
    el.addEventListener("pointerup", finish);
    el.addEventListener("pointercancel", cancel);
  };

  const remove = async () => {
    try { await api.removeBlock(block.id); } catch (err) { toast.error(err instanceof Error ? err.message : "Could not remove that block."); }
    await qc.invalidateQueries({ queryKey: keys.plan(date) });
  };

  const overlapsAnother = others.some((o) => o.id !== block.id && shown.start < o.start + o.duration && o.start < shown.start + shown.duration);
  const short = shown.duration <= 30;

  return (
    <div
      role="group"
      aria-label={`${block.taskTitle}, ${fmtClock(shown.start)} to ${fmtClock(shown.start + shown.duration)}`}
      onPointerDown={begin("move")}
      className={cn(
        "group absolute z-10 cursor-grab touch-none select-none overflow-hidden rounded-lg border border-primary/40 bg-primary/15 px-2 text-xs shadow-sm active:cursor-grabbing",
        block.done && "border-border bg-muted text-muted-foreground line-through",
        preview && "ring-2 ring-primary/40",
        overlapsAnother && "border-destructive/60 bg-destructive/10",
      )}
      style={{ top: shown.start * PX_PER_MIN + 1, height: shown.duration * PX_PER_MIN - 2, left: GUTTER, right: 8 }}
    >
      <div className={cn("flex items-start gap-1", short ? "pt-0.5" : "pt-1.5")}>
        <div className="min-w-0 flex-1">
          <p className="truncate font-medium leading-tight">{block.taskTitle}</p>
          {!short && <p className="mt-0.5 tabular-nums text-muted-foreground">{fmtClock(shown.start)}–{fmtClock(shown.start + shown.duration)} · {fmtDuration(shown.duration)}</p>}
        </div>
        <button
          type="button"
          aria-label={`Remove block for ${block.taskTitle}`}
          onPointerDown={(e) => e.stopPropagation()}
          onClick={remove}
          className="rounded p-0.5 opacity-0 transition-opacity hover:bg-background/60 focus-visible:opacity-100 group-hover:opacity-100"
        >
          <XIcon className="size-3" />
        </button>
      </div>
      <div
        role="separator"
        aria-orientation="horizontal"
        aria-label="Resize block"
        onPointerDown={begin("resize")}
        className="absolute inset-x-0 bottom-0 h-2 cursor-ns-resize"
      />
    </div>
  );
}
