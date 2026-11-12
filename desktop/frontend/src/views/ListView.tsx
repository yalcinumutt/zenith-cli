import { useMemo } from "react";
import type { Task } from "../api/client";
import { NewTask } from "../components/NewTask";
import { TaskSections, type Section } from "../components/TaskSections";
import { upcomingTitle } from "../lib/dates";
import { useInfo, useList, useToday } from "../state/queries";
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
  return (
    <>
      <h1 className="mb-6 flex items-baseline gap-3 text-3xl font-semibold tracking-tight">
        {LIST_TITLES[list]}
        {list === "today" && today && <span className="text-sm font-normal text-muted-foreground">{new Date(`${today}T00:00:00`).toLocaleDateString(undefined, { weekday: "long", day: "numeric", month: "long" })}</span>}
      </h1>
      {list === "today" ? <TodaySections /> : <PlainList list={list} today={today} />}
      {list !== "logbook" && <NewTask />}
    </>
  );
}

function TodaySections() {
  const { data } = useToday();
  const sections: Section[] = useMemo(() => [
    { key: "day", tasks: data?.day ?? [], evening: false },
    { key: "evening", title: "🌙 This Evening", tasks: data?.evening ?? [], evening: true, empty: "Drag a task here to plan it for the evening." },
  ], [data]);
  // The evening heading is always drawn so a task can be dropped onto it.
  return <TaskSections sections={sections} showProject emptyText={EMPTY.today} />;
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
