import { FolderOpen, Pencil, Trash2 } from "lucide-react";
import { useMemo } from "react";
import { api } from "../api/client";
import { attempt, goTo, newProject, promptName } from "../commands/actions";
import { NewTask } from "../components/NewTask";
import { TaskSections, type Section } from "../components/TaskSections";
import { Button } from "../components/ui";
import { useAreaView } from "../state/queries";
import { useUI } from "../state/ui";

/** An area page: its own top-level tasks and the projects inside it. */
export function AreaView({ id }: { id: number }) {
  const { data, error } = useAreaView(id);
  const sections: Section[] = useMemo(() => [{ key: "area", tasks: data?.tasks ?? [] }], [data]);
  if (error) return <p className="py-6 text-sm text-danger">{error instanceof Error ? error.message : "Could not load the area."}</p>;
  if (!data) return null;
  const { area } = data;
  return (
    <>
      <div className="mb-6 flex items-start justify-between gap-4">
        <h1 className="text-3xl font-semibold tracking-tight">{area.name}</h1>
        <div className="flex shrink-0 gap-1 pt-2">
          <Button variant="ghost" onClick={() => newProject(id)}>New project</Button>
          <Button variant="ghost" aria-label="Rename area" onClick={() => promptName({ title: "Rename area", label: "Area name", initial: area.name, confirm: "Rename" }, (name) => void attempt(() => api.renameArea(id, name)))}><Pencil className="h-3.5 w-3.5" aria-hidden /></Button>
          <Button variant="ghost" aria-label="Delete area" onClick={() => useUI.getState().openDialog({ kind: "confirm", title: "Delete area", message: `Delete the area “${area.name}”? Its projects become standalone and its tasks unfiled; nothing else is deleted.`, confirm: "Delete", danger: true, onConfirm: () => void attempt(async () => { await api.deleteArea(id); goTo({ kind: "list", list: "today" }); }) })}><Trash2 className="h-3.5 w-3.5" aria-hidden /></Button>
        </div>
      </div>
      {area.projects.length > 0 && (
        <ul className="mb-6 space-y-0.5" aria-label="Projects in this area">
          {area.projects.map((p) => (
            <li key={p.id}>
              <button onClick={() => goTo({ kind: "project", id: p.id })} className="flex w-full items-center gap-2.5 rounded-md px-3 py-2 text-left text-[15px] hover:bg-hover">
                <FolderOpen className="h-4 w-4 text-accent" aria-hidden /><span className="flex-1">{p.name}</span>
                {p.openTasks > 0 && <span className="text-xs text-muted">{p.openTasks}</span>}
              </button>
            </li>
          ))}
        </ul>
      )}
      <TaskSections sections={sections} showProject emptyText="No tasks directly in this area." />
      <NewTask />
    </>
  );
}
