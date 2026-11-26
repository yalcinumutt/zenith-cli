import { useMemo, useState } from "react";
import { DndContext, DragOverlay, PointerSensor, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { useQueryClient } from "@tanstack/react-query";
import { CheckIcon, ChevronLeftIcon, ChevronRightIcon, SettingsIcon, SparklesIcon, WandSparklesIcon, XIcon } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { api, type Suggestion, type Task } from "../api/client";
import { DayTimeline, TIMELINE_DROP_ID } from "../components/plan/DayTimeline";
import { PlanSettingsDialog } from "../components/plan/PlanSettingsDialog";
import { PlanTasks } from "../components/plan/PlanTasks";
import { addDays, parseDay } from "../lib/dates";
import { clamp, DAY_MIN, DEFAULT_BLOCK, firstFreeSlot, fmtDuration, PX_PER_MIN, snap } from "../lib/plan";
import { keys, useInfo, usePlan } from "../state/queries";

/** Daily Planning: today's tasks on the left, the day as a timeline on the right. */
export function PlanView() {
  const today = useInfo().data?.today ?? "";
  const [picked, setPicked] = useState<string | null>(null);
  const date = picked ?? today;
  const { data: plan, error } = usePlan(date);
  const qc = useQueryClient();
  const [dragging, setDragging] = useState<Task | null>(null);
  const [suggestion, setSuggestion] = useState<{ date: string; value: Suggestion } | null>(null);
  const [working, setWorking] = useState<"quick" | "ai" | "apply" | null>(null);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const sensors = useSensors(useSensor(PointerSensor, { activationConstraint: { distance: 5 } }));

  const nowMin = useMemo(() => {
    if (date !== today) return null;
    const n = new Date();
    return n.getHours() * 60 + n.getMinutes();
  }, [date, today]);

  // A suggestion belongs to the day it was made for; leaving that day drops it.
  const review = suggestion && suggestion.date === date ? suggestion.value : null;

  const refresh = () => qc.invalidateQueries({ queryKey: keys.plan(date) });
  const fail = (err: unknown, fallback: string) => toast.error(err instanceof Error ? err.message : fallback);

  if (error) return <p className="py-6 text-sm text-destructive">{error instanceof Error ? error.message : "Could not load the plan."}</p>;
  if (!plan || !date) return null;

  const total = plan.tasks.reduce((n, t) => n + t.estimate, 0);
  const placed = plan.blocks.reduce((n, b) => n + b.duration, 0);
  const lengthFor = (t: Task) => t.estimate > 0 ? t.estimate : DEFAULT_BLOCK;

  const setEstimate = async (t: Task, minutes: number) => {
    try { await api.updateTask(t.id, { estimate: minutes }); } catch (err) { fail(err, "Could not set the estimate."); }
    await refresh();
  };

  const place = async (t: Task, start?: number) => {
    const duration = lengthFor(t);
    const from = start ?? (nowMin !== null ? nowMin : 9 * 60);
    const at = start ?? firstFreeSlot(plan.blocks, duration, from);
    if (at === null) { toast.error("There is no free slot left that day."); return; }
    try { await api.addBlock(t.id, date, at, duration); } catch (err) { fail(err, "Could not place that task."); }
    await refresh();
  };

  const suggest = async (useAI: boolean) => {
    setWorking(useAI ? "ai" : "quick");
    try {
      setSuggestion({ date, value: await api.suggestPlan(date, useAI) });
    } catch (err) {
      fail(err, "Could not plan the day.");
    } finally {
      setWorking(null);
    }
  };

  const accept = async () => {
    if (!review) return;
    setWorking("apply");
    try {
      await api.applySuggestion(date, review.proposals);
      setSuggestion(null);
      toast.success(`Placed ${review.proposals.length} ${review.proposals.length === 1 ? "block" : "blocks"}`);
    } catch (err) {
      fail(err, "Could not apply the plan.");
    } finally {
      setWorking(null);
    }
    await refresh();
  };

  const onDragEnd = (e: DragEndEvent) => {
    setDragging(null);
    const taskId = e.active.data.current?.taskId as number | undefined;
    const task = plan.tasks.find((t) => t.id === taskId);
    const dragged = e.active.rect.current.translated;
    if (!task || !dragged || e.over?.id !== TIMELINE_DROP_ID) return;
    const start = clamp(snap((dragged.top - e.over.rect.top) / PX_PER_MIN), 0, DAY_MIN - lengthFor(task));
    void place(task, start);
  };

  const label = parseDay(date).toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long" });

  return (
    <DndContext sensors={sensors} onDragStart={(e) => setDragging(plan.tasks.find((t) => t.id === e.active.data.current?.taskId) ?? null)} onDragEnd={onDragEnd} onDragCancel={() => setDragging(null)}>
      <div className="mb-6 flex items-center gap-3">
        <h1 className="text-3xl font-semibold tracking-tight">Daily planning</h1>
        <div className="ml-auto flex items-center gap-1">
          <Button variant="ghost" size="icon" aria-label="Planner settings" onClick={() => setSettingsOpen(true)}><SettingsIcon /></Button>
          <Button variant="ghost" size="icon" aria-label="Previous day" onClick={() => setPicked(addDays(date, -1))}><ChevronLeftIcon /></Button>
          <Button variant="outline" size="sm" disabled={date === today} onClick={() => setPicked(null)}>Today</Button>
          <Button variant="ghost" size="icon" aria-label="Next day" onClick={() => setPicked(addDays(date, 1))}><ChevronRightIcon /></Button>
        </div>
      </div>
      <p className="-mt-4 mb-4 text-sm text-muted-foreground">
        {label} · {fmtDuration(placed) || "0m"} blocked{total > 0 && <> of {fmtDuration(total)} estimated</>}
      </p>
      {plan.calendarError && (
        <p role="status" className="mb-3 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2 text-xs text-destructive">
          Calendar: {plan.calendarError}{plan.busy.length > 0 ? " Showing the last events Zenith could read." : ""}
        </p>
      )}
      <div className="mb-4 flex flex-wrap items-center gap-2">
        <Button size="sm" variant="outline" disabled={working !== null || plan.tasks.length === 0} onClick={() => void suggest(false)}>
          <WandSparklesIcon data-icon="inline-start" />{working === "quick" ? "Planning…" : "Auto-plan"}
        </Button>
        {plan.aiReady ? (
          <Button size="sm" variant="outline" disabled={working !== null || plan.tasks.length === 0} onClick={() => void suggest(true)}
            title={plan.aiCloud ? "Sends task titles, estimates and deadlines to your AI provider" : "Runs on this computer"}>
            <SparklesIcon data-icon="inline-start" />{working === "ai" ? "Thinking…" : "Plan with AI"}
          </Button>
        ) : (
          <Button size="sm" variant="ghost" onClick={() => setSettingsOpen(true)}><SparklesIcon data-icon="inline-start" />Set up AI</Button>
        )}
        {!plan.calendarConfigured && <Button size="sm" variant="ghost" onClick={() => setSettingsOpen(true)}>Connect calendar</Button>}
      </div>
      {review && (
        <div role="region" aria-label="Suggested plan" className="mb-4 rounded-xl border border-primary/40 bg-primary/5 p-3">
          <p className="text-sm font-medium">{review.proposals.length > 0 ? `${review.proposals.length} suggested ${review.proposals.length === 1 ? "block" : "blocks"}` : "Nothing to place"}{review.source === "ai" && " · by AI"}</p>
          {review.summary && <p className="mt-0.5 text-sm text-muted-foreground">{review.summary}</p>}
          {review.unplaced.length > 0 && <p className="mt-1 text-xs text-muted-foreground">No room for: {review.unplaced.join(", ")}</p>}
          {review.dropped.length > 0 && (
            <details className="mt-1 text-xs text-muted-foreground">
              <summary className="cursor-pointer">{review.dropped.length} AI {review.dropped.length === 1 ? "suggestion was" : "suggestions were"} left out</summary>
              <ul className="mt-1 list-disc pl-4">{review.dropped.map((d, i) => <li key={i}>{d}</li>)}</ul>
            </details>
          )}
          <div className="mt-2 flex gap-2">
            <Button size="sm" disabled={working !== null || review.proposals.length === 0} onClick={() => void accept()}><CheckIcon data-icon="inline-start" />Accept</Button>
            <Button size="sm" variant="ghost" disabled={working === "apply"} onClick={() => setSuggestion(null)}><XIcon data-icon="inline-start" />Discard</Button>
          </div>
        </div>
      )}
      <div className="grid grid-cols-[minmax(0,5fr)_minmax(0,6fr)] gap-6">
        <section aria-label="Tasks">
          <PlanTasks tasks={plan.tasks} blocks={plan.blocks} onEstimate={setEstimate} onPlace={(t) => void place(t)} />
        </section>
        <section aria-label="Timeline" className="h-[calc(100vh-19rem)] min-h-96">
          <DayTimeline date={date} blocks={plan.blocks} busy={plan.busy} proposals={review?.proposals ?? []} workStart={plan.workStart} workEnd={plan.workEnd} nowMin={nowMin} />
        </section>
      </div>
      <PlanSettingsDialog open={settingsOpen} onOpenChange={setSettingsOpen} />
      <DragOverlay>
        {dragging && <div className="rounded-lg border bg-card px-3 py-2 text-sm font-medium shadow-lg">{dragging.title}</div>}
      </DragOverlay>
    </DndContext>
  );
}
