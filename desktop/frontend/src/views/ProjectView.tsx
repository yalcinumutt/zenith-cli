import { CheckIcon, FolderPlusIcon, PencilIcon, Trash2Icon } from "lucide-react";
import { useMemo } from "react";
import { api } from "../api/client";
import { attempt, goTo, newHeading, promptName } from "../commands/actions";
import { NewTask } from "../components/NewTask";
import { TaskSections, type Section } from "../components/TaskSections";
import { Button } from "@/components/ui/button";
import { useProjectView } from "../state/queries";
import { useUI } from "../state/ui";

/** A project page: tasks grouped under small divider headings. */
export function ProjectView({ id }: { id: number }) {
  const { data, error } = useProjectView(id);

  const sections: Section[] = useMemo(() => {
    if (!data) return [];
    const topLevel = data.tasks.filter((t) => t.parentId === null || !data.tasks.some((p) => p.id === t.parentId));
    const loose: Section = { key: "h:none", tasks: topLevel.filter((t) => t.headingId === null), heading: null };
    const headed = data.headings.map<Section>((h) => ({
      key: `h:${h.id}`,
      heading: h.id,
      title: h.title,
      empty: "Drop tasks here.",
      tasks: topLevel.filter((t) => t.headingId === h.id),
      actions: (
        <span className="flex gap-1">
          <Button variant="ghost" size="icon-xs" aria-label={`Rename heading ${h.title}`} onClick={() => promptName({ title: "Rename heading", label: "Heading", initial: h.title, confirm: "Rename" }, (title) => void attempt(() => api.renameHeading(h.id, title)))}><PencilIcon /></Button>
          <Button variant="ghost" size="icon-xs" aria-label={`Delete heading ${h.title}`} onClick={() => useUI.getState().openDialog({ kind: "confirm", title: "Delete heading", message: `Delete “${h.title}” and the tasks under it?`, confirm: "Delete", danger: true, onConfirm: () => void attempt(() => api.deleteHeading(h.id)) })}><Trash2Icon /></Button>
        </span>
      ),
    }));
    return [loose, ...headed];
  }, [data]);

  if (error) return <p className="py-6 text-sm text-destructive">{error instanceof Error ? error.message : "Could not load the project."}</p>;
  if (!data) return null;
  const { project } = data;

  return (
    <>
      <div className="mb-6 flex items-start justify-between gap-4">
        <h1 className="text-3xl font-semibold tracking-tight">{project.name}</h1>
        <div className="flex shrink-0 gap-1 pt-2">
          <Button variant="ghost" size="sm" onClick={() => newHeading(id)}><FolderPlusIcon data-icon="inline-start" />Heading</Button>
          <Button variant="ghost" size="icon-sm" aria-label="Rename project" onClick={() => promptName({ title: "Rename project", label: "Project name", initial: project.name, confirm: "Rename" }, (name) => void attempt(() => api.renameProject(id, name)))}><PencilIcon /></Button>
          <Button variant="ghost" size="icon-sm" aria-label="Complete project" onClick={() => void attempt(async () => { await api.completeProject(id); goTo({ kind: "list", list: "today" }); })}><CheckIcon /></Button>
          <Button variant="ghost" size="icon-sm" aria-label="Delete project" onClick={() => useUI.getState().openDialog({ kind: "confirm", title: "Delete project", message: `Delete “${project.name}” with its headings and every task in it? This can’t be undone.`, confirm: "Delete", danger: true, onConfirm: () => void attempt(async () => { await api.deleteProject(id); goTo({ kind: "list", list: "today" }); }) })}><Trash2Icon /></Button>
        </div>
      </div>
      <TaskSections sections={sections} showWhen emptyText="No tasks in this project yet." />
      <NewTask />
    </>
  );
}
