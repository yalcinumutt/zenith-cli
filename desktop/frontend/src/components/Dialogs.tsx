import { useEffect, useRef, useState } from "react";
import { useUI } from "../state/ui";
import { useInfo } from "../state/queries";
import { Button, Modal } from "./ui";

const SPONSORS_URL = "https://github.com/sponsors/yalcinumutt";
const REPO_URL = "https://github.com/yalcinumutt/zenith-cli";

/** Renders whichever dialog the UI store has open: a name prompt, a confirmation, or About. */
export function Dialogs() {
  const dialog = useUI((s) => s.dialog);
  const close = () => useUI.getState().openDialog(null);
  if (!dialog) return null;
  if (dialog.kind === "prompt") return <PromptDialog key={dialog.title} dialog={dialog} close={close} />;
  if (dialog.kind === "confirm") {
    return (
      <Modal open onOpenChange={(o) => !o && close()} title={dialog.title} description={dialog.message}>
        <div className="flex justify-end gap-2">
          <Button onClick={close}>Cancel</Button>
          <Button variant={dialog.danger ? "danger" : "primary"} onClick={() => { close(); dialog.onConfirm(); }}>{dialog.confirm}</Button>
        </div>
      </Modal>
    );
  }
  return <About close={close} />;
}

function PromptDialog({ dialog, close }: { dialog: Extract<NonNullable<ReturnType<typeof useUI.getState>["dialog"]>, { kind: "prompt" }>; close: () => void }) {
  const [value, setValue] = useState(dialog.initial ?? "");
  const ref = useRef<HTMLInputElement>(null);
  useEffect(() => { ref.current?.focus(); ref.current?.select(); }, []);
  const submit = () => {
    if (!value.trim()) return;
    close();
    dialog.onSubmit(value.trim());
  };
  return (
    <Modal open onOpenChange={(o) => !o && close()} title={dialog.title}>
      <form onSubmit={(e) => { e.preventDefault(); submit(); }}>
        <label className="block text-[13px] text-muted" htmlFor="prompt-input">{dialog.label}</label>
        <input id="prompt-input" ref={ref} value={value} onChange={(e) => setValue(e.target.value)}
          className="mt-1 w-full rounded-md border border-line bg-bg px-3 py-2 text-[15px] outline-none focus-visible:ring-2 focus-visible:ring-accent" />
        <div className="mt-4 flex justify-end gap-2">
          <Button type="button" onClick={close}>Cancel</Button>
          <Button type="submit" variant="primary" disabled={!value.trim()}>{dialog.confirm}</Button>
        </div>
      </form>
    </Modal>
  );
}

function About({ close }: { close: () => void }) {
  const info = useInfo().data;
  return (
    <Modal open onOpenChange={(o) => !o && close()} title="About Zenith" description="Tasks, projects and time in one calm place.">
      <div className="space-y-3 text-sm">
        <p>
          <span className="mr-2 rounded-full bg-selected px-2 py-0.5 text-[11px] font-semibold uppercase tracking-wide text-accent">Beta</span>
          Version {info?.version ?? "…"}
        </p>
        <p className="text-muted">
          Zenith is free and open source (MIT). This is a beta: expect rough edges. Reminders only fire while this
          app or <code>zenith daemon</code> is running.
        </p>
        <p>
          If Zenith is useful to you, please consider{" "}
          <a className="text-accent underline" href={SPONSORS_URL} target="_blank" rel="noreferrer">sponsoring the project on GitHub</a>.
          Source and issues: <a className="text-accent underline" href={REPO_URL} target="_blank" rel="noreferrer">github.com/yalcinumutt/zenith-cli</a>.
        </p>
      </div>
      <div className="mt-4 flex justify-end"><Button variant="primary" onClick={close}>Close</Button></div>
    </Modal>
  );
}
