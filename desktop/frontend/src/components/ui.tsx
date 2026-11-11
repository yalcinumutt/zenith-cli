import * as CheckboxPrimitive from "@radix-ui/react-checkbox";
import * as DialogPrimitive from "@radix-ui/react-dialog";
import { Check } from "lucide-react";
import type { ReactNode } from "react";
import { cn } from "../lib/utils";

/** Round task checkbox. Radix gives it the checkbox role; the label names the task. */
export function RoundCheck({ checked, onToggle, label, small }: { checked: boolean; onToggle: () => void; label: string; small?: boolean }) {
  return (
    <CheckboxPrimitive.Root
      checked={checked}
      onCheckedChange={() => onToggle()}
      onClick={(e) => e.stopPropagation()}
      onDoubleClick={(e) => e.stopPropagation()}
      aria-label={label}
      className={cn(
        "mt-0.5 flex shrink-0 items-center justify-center rounded-full border-[1.5px] transition-colors",
        small ? "h-4 w-4" : "h-5 w-5",
        checked ? "border-accent bg-accent text-accent-fg" : "border-muted hover:border-accent",
      )}
    >
      <CheckboxPrimitive.Indicator>
        <Check className={small ? "h-2.5 w-2.5" : "h-3 w-3"} strokeWidth={3} aria-hidden />
      </CheckboxPrimitive.Indicator>
    </CheckboxPrimitive.Root>
  );
}

export function Kbd({ children }: { children: ReactNode }) {
  return <kbd className="rounded border border-line bg-sidebar px-1.5 py-0.5 font-sans text-[11px] text-muted">{children}</kbd>;
}

export function Button({ variant = "default", className, ...props }: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: "default" | "primary" | "danger" | "ghost" }) {
  return (
    <button
      {...props}
      className={cn(
        "rounded-md px-3 py-1.5 text-[13px] font-medium transition-colors disabled:opacity-50",
        variant === "primary" && "bg-accent text-accent-fg hover:opacity-90",
        variant === "danger" && "bg-danger text-bg hover:opacity-90",
        variant === "default" && "border border-line hover:bg-hover",
        variant === "ghost" && "text-muted hover:bg-hover hover:text-fg",
        className,
      )}
    />
  );
}

export function Modal({ open, onOpenChange, title, description, children }: { open: boolean; onOpenChange: (o: boolean) => void; title: string; description?: string; children: ReactNode }) {
  return (
    <DialogPrimitive.Root open={open} onOpenChange={onOpenChange}>
      <DialogPrimitive.Portal>
        <DialogPrimitive.Overlay className="fixed inset-0 z-40 bg-black/30" />
        <DialogPrimitive.Content
          className="pop-in fixed left-1/2 top-[18%] z-50 w-[min(26rem,92vw)] -translate-x-1/2 rounded-xl border border-line bg-card p-5 shadow-2xl"
          aria-describedby={description ? undefined : undefined}
        >
          <DialogPrimitive.Title className="text-base font-semibold">{title}</DialogPrimitive.Title>
          {description ? (
            <DialogPrimitive.Description className="mt-1 text-sm text-muted">{description}</DialogPrimitive.Description>
          ) : (
            <DialogPrimitive.Description className="sr-only">{title}</DialogPrimitive.Description>
          )}
          <div className="mt-4">{children}</div>
        </DialogPrimitive.Content>
      </DialogPrimitive.Portal>
    </DialogPrimitive.Root>
  );
}
