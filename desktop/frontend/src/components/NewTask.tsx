import { Plus } from "lucide-react";
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
    <div className="mt-2 flex items-center gap-3 rounded-lg px-3 py-2 focus-within:bg-hover">
      <Plus className="h-4 w-4 text-muted" aria-hidden />
      <input
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
        className="w-full bg-transparent text-[15px] outline-none placeholder:text-muted"
      />
    </div>
  );
}
