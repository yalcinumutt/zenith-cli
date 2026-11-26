import { useMemo, useState } from "react";
import { ArrowDownWideNarrowIcon, CalendarClockIcon } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { Task } from "../api/client";
import { NewTask } from "../components/NewTask";
import { TaskSections, type Section } from "../components/TaskSections";
import { upcomingTitle } from "../lib/dates";
import { useInfo, useList, useToday } from "../state/queries";
import { goTo } from "../commands/actions";
import { LIST_TITLES, type ListId } from "../state/ui";

const EMPTY: Record<ListId, string> = {
  inbox: "Your inbox is empty.",
  today: "Nothing planned for today.",
  upcoming: "Nothing scheduled ahead.",
  anytime: "Nothing here.",
  someday: "Nothing parked for someday.",
  logbook: "Completed tasks will appear here.",
};

export function ListView({ list }: { list: ListId }) {
  const today = useInfo().data?.today ?? "";
  const [sort, setSort] = useState<TodaySort>(loadSort);
  const toggleSort = () => {
    const next: TodaySort = sort === "priority" ? "manual" : "priority";
    setSort(next);
    try { localStorage.setItem(SORT_KEY, next); } catch { /* not persisted */ }
  };
  return (
    <>
      <h1 className="mb-6 flex items-baseline gap-3 text-3xl font-semibold tracking-tight">
        {LIST_TITLES[list]}
        {list === "today" && today && <span className="text-sm font-normal text-muted-foreground">{new Date(`${today}T00:00:00`).toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long" })}</span>}
        {list === "today" && (
          <Button variant="outline" size="sm" className="ml-auto" onClick={() => goTo({ kind: "plan" })}>
            <CalendarClockIcon data-icon="inline-start" />Daily planning
          </Button>
        )}
        {list === "today" && (
          <Button variant="ghost" size="sm" aria-pressed={sort === "priority"} onClick={toggleSort}>
            <ArrowDownWideNarrowIcon data-icon="inline-start" />{sort === "priority" ? "Sorted by priority" : "Sort by priority"}
          </Button>
        )}
      </h1>
      {list === "today" ? <TodaySections sort={sort} /> : <PlainList list={list} today={today} />}
      {list !== "logbook" && <NewTask />}
    </>
  );
}

type TodaySort = "manual" | "priority";
const SORT_KEY = "zenith.todaySort";
const loadSort = (): TodaySort => { try { return localStorage.getItem(SORT_KEY) === "priority" ? "priority" : "manual"; } catch { return "manual"; } };

/** Highest priority first; a stable sort, so equal priorities keep their manual order. */
const byPriority = (tasks: Task[]): Task[] => [...tasks].sort((a, b) => b.priority - a.priority);

/** Today's day tasks grouped under their project (no project last), then the evening block. */
function TodaySections({ sort }: { sort: TodaySort }) {
  const { data } = useToday();
  const sections: Section[] = useMemo(() => {
    const order = (ts: Task[]) => (sort === "priority" ? byPriority(ts) : ts);
    const groups = new Map<number | null, Task[]>();
    for (const t of data?.day ?? []) groups.set(t.projectId ?? null, [...(groups.get(t.projectId ?? null) ?? []), t]);
    const day: Section[] = [...groups]
      .sort(([a, ta], [b, tb]) => (a === null ? 1 : 0) - (b === null ? 1 : 0) || (ta[0].projectName ?? "").localeCompare(tb[0].projectName ?? ""))
      .map(([id, ts]) => ({ key: `day:${id ?? "none"}`, title: id === null ? "No project" : ts[0].projectName, tasks: order(ts), evening: false }));
    return [
      ...(day.length ? day : [{ key: "day", tasks: [], evening: false }]),
      { key: "evening", title: "🌙 This Evening", tasks: order(data?.evening ?? []), evening: true, empty: "Drag a task here to plan it for the evening." },
    ];
  }, [data, sort]);
  // The evening heading is always drawn so a task can be dropped onto it.
  return <TaskSections sections={sections} emptyText={EMPTY.today} />;
}

function PlainList({ list, today }: { list: ListId; today: string }) {
  const { data } = useList(list);
  const sections: Section[] = useMemo(() => {
    const tasks = data ?? [];
    if (list === "upcoming") {
      const byDay = new Map<string, Task[]>();
      for (const t of tasks) byDay.set(t.whenDate, [...(byDay.get(t.whenDate) ?? []), t]);
      return [...byDay].map(([day, ts]) => ({ key: day, title: upcomingTitle(day, today), tasks: ts }));
    }
    return [{ key: "all", tasks }];
  }, [data, list, today]);
  return <TaskSections sections={sections} showProject showWhen={list !== "upcoming"} emptyText={EMPTY[list]} />;
}
