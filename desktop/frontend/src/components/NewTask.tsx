import { PlusIcon } from "lucide-react";
import { Input } from "@/components/ui/input";
import { useEffect, useRef, useState } from "react";
import { createTask } from "../commands/actions";
import { useInfo } from "../state/queries";
import { useUI } from "../state/ui";

/** The single "New to-do" line. Mod+N focuses it from anywhere. */
export function NewTask() {
  const view = useUI((s) => s.view);
  const focusTick = useUI((s) => s.newTaskFocus);
  const today = useInfo().data?.today ?? "";
  const [text, setText] = useState("");
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => { if (focusTick > 0) ref.current?.focus(); }, [focusTick]);

  return (
    <div className="mt-2 flex items-center gap-3 rounded-lg px-3 py-2 focus-within:bg-muted">
      <PlusIcon className="size-4 text-muted-foreground" aria-hidden />
      <Input
        ref={ref}
        aria-label="New to-do"
        placeholder="New to-do"
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={async (e) => {
          if (e.key === "Escape") { (e.target as HTMLInputElement).blur(); return; }
          if (e.key === "Enter" && text.trim() && today) {
            const t = text;
            setText("");
            await createTask(t, view, today);
          }
        }}
        className="h-8 border-transparent bg-transparent px-0 text-[15px] shadow-none focus-visible:ring-0 dark:bg-transparent"
      />
    </div>
  );
}
