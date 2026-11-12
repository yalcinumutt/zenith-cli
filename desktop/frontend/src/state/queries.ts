import { QueryClient, useQuery } from "@tanstack/react-query";
import { api, onChange } from "../api/client";

export const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 5_000, refetchOnWindowFocus: true, retry: false } },
});

// One cache key per list, so core events can invalidate precisely.
export const keys = {
  info: ["info"] as const,
  counts: ["counts"] as const,
  sidebar: ["sidebar"] as const,
  tags: ["tags"] as const,
  today: ["today"] as const,
  list: (kind: string) => ["list", kind] as const,
  project: (id: number) => ["project", id] as const,
  area: (id: number) => ["area", id] as const,
  subtasks: (id: number) => ["subtasks", id] as const,
  reminders: ["reminders"] as const,
  habits: ["habits"] as const,
};

export const useInfo = () => useQuery({ queryKey: keys.info, queryFn: api.info, staleTime: 30_000, refetchInterval: 60_000 });
export const useCounts = () => useQuery({ queryKey: keys.counts, queryFn: api.counts });
export const useSidebar = () => useQuery({ queryKey: keys.sidebar, queryFn: api.sidebar });
export const useTags = () => useQuery({ queryKey: keys.tags, queryFn: api.tags });
export const useToday = () => useQuery({ queryKey: keys.today, queryFn: api.today });
export const useList = (kind: string) => useQuery({ queryKey: keys.list(kind), queryFn: () => api.list(kind) });
export const useProjectView = (id: number) => useQuery({ queryKey: keys.project(id), queryFn: () => api.projectView(id) });
export const useAreaView = (id: number) => useQuery({ queryKey: keys.area(id), queryFn: () => api.areaView(id) });
export const useSubtasks = (id: number, enabled: boolean) =>
  useQuery({ queryKey: keys.subtasks(id), queryFn: () => api.subtasks(id), enabled });
export const useHabits = () => useQuery({ queryKey: keys.habits, queryFn: api.habits });
export const usePendingReminders = () => useQuery({ queryKey: keys.reminders, queryFn: api.pendingReminders });

/**
 * Wire core events to the cache: a task edit or list invalidation refreshes every
 * query derived from tasks; a write from another process (the CLI) refreshes all.
 * Reminder firings refresh the reminder list only.
 */
export function startChangeBridge(): () => void {
  let timer: ReturnType<typeof setTimeout> | undefined;
  const refresh = (what: "all" | "reminders") => {
    clearTimeout(timer);
    timer = setTimeout(() => {
      if (what === "reminders") void queryClient.invalidateQueries({ queryKey: keys.reminders });
      else void queryClient.invalidateQueries();
    }, 40); // coalesce the burst of events one change produces
  };
  const off = onChange((e) => refresh(e.type === "reminder" ? "reminders" : "all"));
  return () => {
    clearTimeout(timer);
    off();
  };
}
