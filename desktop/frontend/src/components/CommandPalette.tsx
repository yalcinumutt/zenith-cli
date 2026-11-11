import * as DialogPrimitive from "@radix-ui/react-dialog";
import { Command as Cmdk } from "cmdk";
import { commands } from "../commands/registry";
import { goTo } from "../commands/actions";
import { formatShortcut } from "../lib/platform";
import { useSidebar } from "../state/queries";
import { useUI } from "../state/ui";
import { Kbd } from "./ui";

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
  const groups = ["Navigate", "Task", "Organise", "App"] as const;
  const projects = [...(sidebar?.areas.flatMap((a) => a.projects) ?? []), ...(sidebar?.standalone ?? [])];

  return (
    <DialogPrimitive.Root open={open} onOpenChange={setOpen}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-40 bg-black/30" />
        <DialogPrimitive.Content className="pop-in fixed left-1/2 top-[14%] z-50 w-[min(34rem,92vw)] -translate-x-1/2 overflow-hidden rounded-xl border border-line bg-card shadow-2xl">
          <DialogPrimitive.Title className="sr-only">Command palette</DialogPrimitive.Title>
          <DialogPrimitive.Description className="sr-only">Type to find a list, project or action.</DialogPrimitive.Description>
          <Cmdk label="Command palette" loop>
            <Cmdk.Input autoFocus placeholder="Type a command, list or project…" className="w-full border-b border-line bg-transparent px-4 py-3 text-[15px] outline-none" />
            <Cmdk.List className="max-h-[22rem] overflow-auto p-2">
              <Cmdk.Empty className="px-3 py-6 text-center text-sm text-muted">No matches.</Cmdk.Empty>
              {(projects.length > 0 || (sidebar?.areas.length ?? 0) > 0) && (
                <Cmdk.Group heading="Projects and areas" className="text-[11px] font-semibold uppercase tracking-wide text-muted [&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:py-1.5">
                  {sidebar?.areas.map((a) => (
                    <Item key={`a${a.id}`} value={`area ${a.name}`} onSelect={() => run(() => goTo({ kind: "area", id: a.id }))}>Area: {a.name}</Item>
                  ))}
                  {projects.map((p) => (
                    <Item key={`p${p.id}`} value={`project ${p.name}`} onSelect={() => run(() => goTo({ kind: "project", id: p.id }))}>Project: {p.name}</Item>
                  ))}
                </Cmdk.Group>
              )}
              {groups.map((g) => {
                const items = available.filter((c) => c.group === g);
                if (items.length === 0) return null;
                return (
                  <Cmdk.Group key={g} heading={g} className="text-[11px] font-semibold uppercase tracking-wide text-muted [&_[cmdk-group-heading]]:px-3 [&_[cmdk-group-heading]]:py-1.5">
                    {items.map((c) => (
                      <Item key={c.id} value={`${c.title} ${(c.keywords ?? []).join(" ")}`} onSelect={() => run(c.run)}>
                        <span>{c.title}</span>
                        {c.shortcut && <Kbd>{formatShortcut(c.shortcut)}</Kbd>}
                      </Item>
                    ))}
                  </Cmdk.Group>
                );
              })}
            </Cmdk.List>
          </Cmdk>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}

function Item({ children, onSelect, value }: { children: React.ReactNode; onSelect: () => void; value: string }) {
  return (
    <Cmdk.Item value={value} onSelect={onSelect}
      className="flex cursor-default items-center justify-between gap-3 rounded-md px-3 py-2 text-sm normal-case tracking-normal text-fg data-[selected=true]:bg-selected">
      {children}
    </Cmdk.Item>
  );
}
