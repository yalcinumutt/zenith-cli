import { useEffect, useRef, useState } from "react";
import {
  AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription,
  AlertDialogFooter, AlertDialogHeader, AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Calendar } from "@/components/ui/calendar";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { fmtISO, parseDay } from "../lib/dates";
import { useInfo } from "../state/queries";
import { useUI, type Dialog as DialogState } from "../state/ui";

const SPONSORS_URL = "https://github.com/sponsors/yalcinumutt";
const REPO_URL = "https://github.com/yalcinumutt/zenith-cli";

type Of<K extends DialogState["kind"]> = Extract<DialogState, { kind: K }>;

/** Renders whichever dialog the UI store has open: a name prompt, a date, a confirmation, or About. */
export function Dialogs() {
  const dialog = useUI((s) => s.dialog);
  const close = () => useUI.getState().openDialog(null);
  if (!dialog) return null;
  if (dialog.kind === "prompt") return <PromptDialog key={dialog.title} dialog={dialog} close={close} />;
  if (dialog.kind === "date") return <DateDialog key={dialog.title} dialog={dialog} close={close} />;
  if (dialog.kind === "confirm") return <ConfirmDialog dialog={dialog} close={close} />;
  return <About close={close} />;
}

function ConfirmDialog({ dialog, close }: { dialog: Of<"confirm">; close: () => void }) {
  return (
    <AlertDialog open onOpenChange={(o) => !o && close()}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{dialog.title}</AlertDialogTitle>
          <AlertDialogDescription>{dialog.message}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction variant={dialog.danger ? "destructive" : "default"} onClick={() => { close(); dialog.onConfirm(); }}>{dialog.confirm}</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

function PromptDialog({ dialog, close }: { dialog: Of<"prompt">; close: () => void }) {
  const [value, setValue] = useState(dialog.initial ?? "");
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => { ref.current?.select(); }, []);
  const submit = () => {
    if (!value.trim()) return;
    close();
    dialog.onSubmit(value.trim());
  };
  return (
    <Dialog open onOpenChange={(o) => !o && close()}>
      <DialogContent className="sm:max-w-sm">
        <form onSubmit={(e) => { e.preventDefault(); submit(); }} className="flex flex-col gap-4">
          <DialogHeader>
            <DialogTitle>{dialog.title}</DialogTitle>
            <DialogDescription className="sr-only">{dialog.label}</DialogDescription>
          </DialogHeader>
          <div className="flex flex-col gap-2">
            <Label htmlFor="prompt-input">{dialog.label}</Label>
            <Input id="prompt-input" ref={ref} autoFocus value={value} onChange={(e) => setValue(e.target.value)} />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={close}>Cancel</Button>
            <Button type="submit" disabled={!value.trim()}>{dialog.confirm}</Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function DateDialog({ dialog, close }: { dialog: Of<"date">; close: () => void }) {
  const [value, setValue] = useState(dialog.initial);
  const submit = (d: string) => { close(); dialog.onSubmit(d); };
  return (
    <Dialog open onOpenChange={(o) => !o && close()}>
      <DialogContent className="w-fit sm:max-w-fit">
        <DialogHeader>
          <DialogTitle>{dialog.title}</DialogTitle>
          <DialogDescription className="sr-only">Choose a date</DialogDescription>
        </DialogHeader>
        <Calendar mode="single" required selected={parseDay(value)} defaultMonth={parseDay(value)} onSelect={(d) => d && setValue(fmtISO(d))} className="mx-auto p-0" />
        <DialogFooter>
          <Button variant="outline" onClick={close}>Cancel</Button>
          <Button onClick={() => submit(value)}>{dialog.confirm}</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function About({ close }: { close: () => void }) {
  const info = useInfo().data;
  return (
    <Dialog open onOpenChange={(o) => !o && close()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>About Zenith</DialogTitle>
          <DialogDescription>Tasks, projects and time in one calm place.</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-3 text-sm">
          <p className="flex items-center gap-2"><Badge variant="secondary">Beta</Badge>Version {info?.version ?? "…"}</p>
          <p className="text-muted-foreground">
            Zenith is free and open source (MIT). This is a beta: expect rough edges. Reminders only fire while this
            app or <code>zenith daemon</code> is running.
          </p>
          <p>
            If Zenith is useful to you, please consider{" "}
            <a className="text-primary underline" href={SPONSORS_URL} target="_blank" rel="noreferrer">sponsoring the project on GitHub</a>.
            Source and issues: <a className="text-primary underline" href={REPO_URL} target="_blank" rel="noreferrer">github.com/yalcinumutt/zenith-cli</a>.
          </p>
        </div>
        <DialogFooter><Button onClick={close}>Close</Button></DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
