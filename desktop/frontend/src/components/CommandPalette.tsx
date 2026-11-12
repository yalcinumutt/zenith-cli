import { CommandDialog, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList, CommandShortcut } from "@/components/ui/command";
import { FolderOpenIcon, LayersIcon } from "lucide-react";
import { commands } from "../commands/registry";
import { goTo } from "../commands/actions";
import { formatShortcut } from "../lib/platform";
import { useSidebar } from "../state/queries";
import { useUI } from "../state/ui";

const GROUPS = ["Navigate", "Task", "Organise", "App"] as const;

/** Mod+K palette: finds lists, projects, areas and every registered action by typing. */
export function CommandPalette() {
  const open = useUI((s) => s.paletteOpen);
  const setOpen = useUI((s) => s.setPalette);
  const sidebar = useSidebar().data;

  const run = (fn: () => void) => {
    setOpen(false);
    // Let the dialog close and return focus before the action runs.
    setTimeout(fn, 0);
  };

  const available = open ? commands.filter((c) => c.id !== "palette" && (!c.when || c.when())) : [];
  const projects = [...(sidebar?.areas.flatMap((a) => a.projects) ?? []), ...(sidebar?.standalone ?? [])];

  return (
    <CommandDialog open={open} onOpenChange={setOpen} title="Command palette" description="Type to find a list, project or action.">
      <CommandInput placeholder="Type a command, list or project…" />
      <CommandList>
        <CommandEmpty>No matches.</CommandEmpty>
        {(projects.length > 0 || (sidebar?.areas.length ?? 0) > 0) && (
          <CommandGroup heading="Projects and areas">
            {sidebar?.areas.map((a) => (
              <CommandItem key={`a${a.id}`} value={`area ${a.name}`} onSelect={() => run(() => goTo({ kind: "area", id: a.id }))}>
                <LayersIcon />Area: {a.name}
              </CommandItem>
            ))}
            {projects.map((p) => (
              <CommandItem key={`p${p.id}`} value={`project ${p.name}`} onSelect={() => run(() => goTo({ kind: "project", id: p.id }))}>
                <FolderOpenIcon />Project: {p.name}
              </CommandItem>
            ))}
          </CommandGroup>
        )}
        {GROUPS.map((g) => {
          const items = available.filter((c) => c.group === g);
          if (items.length === 0) return null;
          return (
            <CommandGroup key={g} heading={g}>
              {items.map((c) => (
                <CommandItem key={c.id} value={`${c.title} ${(c.keywords ?? []).join(" ")}`} onSelect={() => run(c.run)}>
                  {c.title}
                  {c.shortcut && <CommandShortcut>{formatShortcut(c.shortcut)}</CommandShortcut>}
                </CommandItem>
              ))}
            </CommandGroup>
          );
        })}
      </CommandList>
    </CommandDialog>
  );
}
