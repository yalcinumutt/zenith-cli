import { CheckCheck, Calendar, FolderOpen, Info, Inbox, Layers, Moon, Plus, Star, SunMoon, type LucideIcon } from "lucide-react";
import { cycleTheme, goTo, newArea, newProject } from "../commands/actions";
import { cn } from "../lib/utils";
import { useCounts, useSidebar } from "../state/queries";
import { LIST_ORDER, LIST_TITLES, useUI, type ListId, type View } from "../state/ui";

const ICONS: Record<ListId, LucideIcon> = { inbox: Inbox, today: Star, upcoming: Calendar, anytime: Layers, someday: Moon, logbook: CheckCheck };
const ICON_TONE: Record<ListId, string> = { inbox: "text-accent", today: "text-amber-600 dark:text-amber-400", upcoming: "text-accent", anytime: "text-accent", someday: "text-muted", logbook: "text-muted" };

function same(a: View, b: View): boolean {
  if (a.kind !== b.kind) return false;
  if (a.kind === "list" && b.kind === "list") return a.list === b.list;
  return a.kind !== "list" && b.kind !== "list" && a.id === b.id;
}

export function Sidebar() {
  const view = useUI((s) => s.view);
  const counts = useCounts().data;
  const sidebar = useSidebar().data;
  const theme = useUI((s) => s.theme);

  const Item = ({ target, label, icon: Icon, tone, count, indent }: { target: View; label: string; icon: LucideIcon; tone?: string; count?: number; indent?: boolean }) => (
    <button
      onClick={() => goTo(target)}
      aria-current={same(view, target) ? "page" : undefined}
      className={cn("flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13px]", indent && "pl-5", same(view, target) ? "bg-selected font-medium" : "hover:bg-hover")}
    >
      <Icon className={cn("h-4 w-4 shrink-0", tone ?? "text-accent")} aria-hidden />
      <span className="flex-1 truncate">{label}</span>
      {!!count && <span className="text-xs text-muted" aria-label={`${count} tasks`}>{count}</span>}
    </button>
  );

  return (
    <aside className="flex w-60 shrink-0 flex-col border-r border-line bg-sidebar" aria-label="Sidebar">
      <nav className="flex-1 space-y-0.5 overflow-auto p-3 pt-6" aria-label="Lists">
        {LIST_ORDER.map((l) => (
          <Item key={l} target={{ kind: "list", list: l }} label={LIST_TITLES[l]} icon={ICONS[l]} tone={ICON_TONE[l]}
            count={l === "inbox" ? counts?.inbox : l === "today" ? counts?.today : l === "upcoming" ? counts?.upcoming : undefined} />
        ))}

        {sidebar?.areas.map((a) => (
          <div key={a.id} className="mt-5">
            <button onClick={() => goTo({ kind: "area", id: a.id })} className="w-full px-2.5 pb-1 text-left text-[11px] font-semibold uppercase tracking-wide text-muted hover:text-fg">{a.name}</button>
            {a.projects.map((p) => <Item key={p.id} target={{ kind: "project", id: p.id }} label={p.name} icon={FolderOpen} count={p.openTasks} />)}
          </div>
        ))}
        {(sidebar?.standalone.length ?? 0) > 0 && (
          <div className="mt-5">
            <div className="px-2.5 pb-1 text-[11px] font-semibold uppercase tracking-wide text-muted">Projects</div>
            {sidebar?.standalone.map((p) => <Item key={p.id} target={{ kind: "project", id: p.id }} label={p.name} icon={FolderOpen} count={p.openTasks} />)}
          </div>
        )}
      </nav>

      <div className="space-y-0.5 border-t border-line p-3">
        <button onClick={() => newProject()} className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13px] text-muted hover:bg-hover hover:text-fg"><Plus className="h-4 w-4" aria-hidden />New project</button>
        <button onClick={newArea} className="flex w-full items-center gap-2.5 rounded-md px-2.5 py-1.5 text-left text-[13px] text-muted hover:bg-hover hover:text-fg"><Plus className="h-4 w-4" aria-hidden />New area</button>
        <div className="flex items-center justify-between pt-2">
          <button onClick={() => useUI.getState().openDialog({ kind: "about" })} className="flex items-center gap-1.5 rounded-md px-2 py-1 text-xs text-muted hover:bg-hover hover:text-fg">
            <Info className="h-3.5 w-3.5" aria-hidden />About
            <span className="rounded-full bg-selected px-1.5 py-px text-[10px] font-semibold uppercase text-accent">Beta</span>
          </button>
          <button onClick={cycleTheme} aria-label={`Theme: ${theme}. Change theme`} title={`Theme: ${theme}`} className="rounded-md p-1.5 text-muted hover:bg-hover hover:text-fg"><SunMoon className="h-4 w-4" aria-hidden /></button>
        </div>
      </div>
    </aside>
  );
}
