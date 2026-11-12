import { CommandDialog, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { FolderOpenIcon, InboxIcon, LayersIcon } from "lucide-react";
import { api } from "../api/client";
import { attempt } from "../commands/actions";
import { useSidebar } from "../state/queries";
import { useUI } from "../state/ui";

/** "Move to…": pick a project, an area, or Inbox for a task. Type to filter, arrows and Enter to choose. */
export function MovePicker() {
  const taskId = useUI((s) => s.moveTaskId);
  const task = useUI((s) => s.rows.find((r) => r.task.id === s.moveTaskId)?.task);
  const sidebar = useSidebar().data;
  const close = () => useUI.getState().setMoveTask(null);
  if (taskId === null || !task) return null;

  const go = (patch: Parameters<typeof api.updateTask>[1]) => {
    close();
    void attempt(() => api.updateTask(task.id, patch));
  };

  return (
    <CommandDialog open onOpenChange={(o) => !o && close()} title={`Move “${task.title}”`} description="Choose a project or area to move the task to.">
      <CommandInput placeholder={`Move “${task.title}” to…`} />
      <CommandList>
        <CommandEmpty>No matches.</CommandEmpty>
        <CommandGroup>
          <CommandItem value="inbox no project unfile" onSelect={() => go({ clearProject: true })}><InboxIcon />Inbox (no project)</CommandItem>
        </CommandGroup>
        {sidebar?.areas.map((a) => (
          <CommandGroup key={a.id} heading={a.name}>
            <CommandItem value={`area ${a.name}`} onSelect={() => go({ areaId: a.id })}><LayersIcon />{a.name} (area, no project)</CommandItem>
            {a.projects.map((p) => (
              <CommandItem key={p.id} value={`${a.name} ${p.name}`} onSelect={() => go({ projectId: p.id })}><FolderOpenIcon />{p.name}</CommandItem>
            ))}
          </CommandGroup>
        ))}
        {(sidebar?.standalone.length ?? 0) > 0 && (
          <CommandGroup heading="Projects">
            {sidebar?.standalone.map((p) => (
              <CommandItem key={p.id} value={p.name} onSelect={() => go({ projectId: p.id })}><FolderOpenIcon />{p.name}</CommandItem>
            ))}
          </CommandGroup>
        )}
      </CommandList>
    </CommandDialog>
  );
}
