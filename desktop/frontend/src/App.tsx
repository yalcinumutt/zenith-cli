import { useEffect } from "react";
import { api } from "./api/client";
import { commands } from "./commands/registry";
import { CommandPalette } from "./components/CommandPalette";
import { Dialogs } from "./components/Dialogs";
import { MovePicker } from "./components/MovePicker";
import { Sidebar } from "./components/Sidebar";
import { Toaster } from "@/components/ui/sonner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { isTyping, matches } from "./lib/platform";
import { parseView, useUI, type Theme } from "./state/ui";
import { AreaView } from "./views/AreaView";
import { ListView } from "./views/ListView";
import { HabitsView } from "./views/HabitsView";
import { ProjectView } from "./views/ProjectView";

/** Apply the theme: the stored override, or the system setting when it is "system". */
function useTheme(theme: Theme) {
  useEffect(() => {
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    const apply = () => document.documentElement.classList.toggle("dark", theme === "dark" || (theme === "system" && media.matches));
    apply();
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [theme]);
}

/** One global key handler, driven entirely by the command registry. */
function useShortcuts() {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.defaultPrevented || e.isComposing) return;
      const ui = useUI.getState();
      const dialogOpen = ui.dialog !== null || ui.paletteOpen || ui.moveTaskId !== null;
      const typing = isTyping(e.target);
      for (const c of commands) {
        if (!c.shortcut || !matches(e, c.shortcut)) continue;
        const plain = !c.shortcut.includes("mod");
        if (plain && (typing || dialogOpen)) continue; // x, space, arrows belong to the field or dialog
        if (c.shortcut === "escape" && dialogOpen) continue;
        if (c.id === "open-task" && typing) continue;
        if (c.when && !c.when()) continue;
        if (dialogOpen && c.id !== "palette") continue;
        e.preventDefault();
        c.run();
        return;
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);
}

export default function App() {
  const view = useUI((s) => s.view);
  const theme = useUI((s) => s.theme);
  useTheme(theme);
  useShortcuts();

  // Restore the last view and theme from the app's own preferences.
  useEffect(() => {
    void api.getPrefs().then((p) => {
      useUI.getState().setTheme(p.theme as Theme);
      useUI.getState().setView(parseView(p.lastView));
    }).catch(() => undefined);
  }, []);

  return (
    <TooltipProvider>
    <div className="flex h-full bg-background text-foreground">
      <Sidebar />
      <main className="flex-1 overflow-auto" aria-label="Tasks" onClick={(e) => { if (e.target === e.currentTarget) useUI.getState().select(null); }}>
        <div className="mx-auto max-w-2xl px-10 pb-40 pt-12">
          {view.kind === "list" && <ListView key={view.list} list={view.list} />}
          {view.kind === "project" && <ProjectView key={`p${view.id}`} id={view.id} />}
          {view.kind === "habits" && <HabitsView />}
          {view.kind === "area" && <AreaView key={`a${view.id}`} id={view.id} />}
        </div>
      </main>
      <CommandPalette />
      <MovePicker />
      <Dialogs />
      <Toaster position="top-center" />
    </div>
    </TooltipProvider>
  );
}
