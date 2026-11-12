import { useState } from "react";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { HabitDetail } from "../components/habits/HabitDetail";
import { InsightsTab } from "../components/habits/InsightsTab";
import { TodayTab } from "../components/habits/TodayTab";
import { useHabits, useInfo } from "../state/queries";

/** Habits: today's checklist, a calendar with insights, and one habit's heatmap history. */
export function HabitsView() {
  const today = useInfo().data?.today ?? "";
  const { data: habits, error } = useHabits();
  const [openId, setOpenId] = useState<number | null>(null);
  const [tab, setTab] = useState("today");

  if (error) return <p className="py-6 text-sm text-destructive">{error instanceof Error ? error.message : "Could not load habits."}</p>;
  if (!habits || !today) return null;

  const open = habits.find((h) => h.id === openId);
  if (open) return <HabitDetail habit={open} today={today} onBack={() => setOpenId(null)} />;

  return (
    <Tabs value={tab} onValueChange={setTab}>
      <div className="mb-6 flex items-center justify-between">
        <h1 className="text-3xl font-semibold tracking-tight">Habits</h1>
        <TabsList>
          <TabsTrigger value="today">Today</TabsTrigger>
          <TabsTrigger value="insights">Insights</TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value="today"><TodayTab habits={habits} today={today} onOpen={setOpenId} /></TabsContent>
      <TabsContent value="insights"><InsightsTab habits={habits} today={today} onOpen={setOpenId} /></TabsContent>
    </Tabs>
  );
}
