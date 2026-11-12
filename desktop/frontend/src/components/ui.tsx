import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";

/** Round task checkbox on shadcn's Checkbox; the label names the task. */
export function RoundCheck({ checked, onToggle, label, small }: { checked: boolean; onToggle: () => void; label: string; small?: boolean }) {
  return (
    <Checkbox
      checked={checked}
      onCheckedChange={() => onToggle()}
      onClick={(e) => e.stopPropagation()}
      onDoubleClick={(e) => e.stopPropagation()}
      aria-label={label}
      className={cn("mt-0.5 rounded-full border-[1.5px] border-muted-foreground hover:border-primary", small ? "size-4" : "size-5")}
    />
  );
}
