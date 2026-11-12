import { FolderOpenIcon, PencilIcon, PlusIcon, Trash2Icon } from "lucide-react";
import { useMemo } from "react";
import { Badge } from "@/components/ui/badge";
import { api } from "../api/client";
import { attempt, goTo, newProject, promptName } from "../commands/actions";
import { NewTask } from "../components/NewTask";
import { TaskSections, type Section } from "../components/TaskSections";
import { Button } from "@/components/ui/button";
import { useAreaView } from "../state/queries";
import { useUI } from "../state/ui";

/** An area page: its own top-level tasks and the projects inside it. */
export function AreaView({ id }: { id: number }) {
  const { data, error } = useAreaView(id);
  const sections: Section[] = useMemo(() => [{ key: "area", tasks: data?.tasks ?? [] }], [data]);
  if (error) return <p className="py-6 text-sm text-destructive">{error instanceof Error ? error.message : "Could not load the area."}</p>;
  if (!data) return null;
  const { area } = data;
  return (
    <>
      <div className="mb-6 flex items-start justify-between gap-4">
        <h1 className="text-3xl font-semibold tracking-tight">{area.name}</h1>
        <div className="flex shrink-0 gap-1 pt-2">
          <Button variant="ghost" size="sm" onClick={() => newProject(id)}><PlusIcon data-icon="inline-start" />New project</Button>
          <Button variant="ghost" size="icon-sm" aria-label="Rename area" onClick={() => promptName({ title: "Rename area", label: "Area name", initial: area.name, confirm: "Rename" }, (name) => void attempt(() => api.renameArea(id, name)))}><PencilIcon /></Button>
          <Button variant="ghost" size="icon-sm" aria-label="Delete area" onClick={() => useUI.getState().openDialog({ kind: "confirm", title: "Delete area", message: `Delete the area “${area.name}”? Its projects become standalone and its tasks unfiled; nothing else is deleted.`, confirm: "Delete", danger: true, onConfirm: () => void attempt(async () => { await api.deleteArea(id); goTo({ kind: "list", list: "today" }); }) })}><Trash2Icon /></Button>
        </div>
      </div>
      {area.projects.length > 0 && (
        <ul className="mb-6 flex flex-col gap-0.5" aria-label="Projects in this area">
          {area.projects.map((p) => (
            <li key={p.id}>
              <Button variant="ghost" onClick={() => goTo({ kind: "project", id: p.id })} className="h-9 w-full justify-start gap-2.5 px-3 text-[15px] font-normal">
                <FolderOpenIcon className="text-primary" aria-hidden /><span className="flex-1 text-left">{p.name}</span>
                {p.openTasks > 0 && <Badge variant="secondary">{p.openTasks}</Badge>}
              </Button>
            </li>
          ))}
        </ul>
      )}
      <TaskSections sections={sections} showProject showWhen emptyText="No tasks directly in this area." />
      <NewTask />
    </>
  );
}
