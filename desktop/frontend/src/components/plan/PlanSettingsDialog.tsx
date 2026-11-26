import { useEffect, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { api, type PlanSettings } from "../../api/client";
import { fromTimeInput, toTimeInput } from "../../lib/plan";
import { keys, usePlanSettings } from "../../state/queries";

const KINDS = [
  { value: "", label: "Off" },
  { value: "local", label: "Local model (Ollama, LM Studio)" },
  { value: "openai", label: "OpenAI" },
  { value: "anthropic", label: "Anthropic" },
];

/** Working hours, the calendar link and the AI provider. Nothing is saved until Save. */
export function PlanSettingsDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { data } = usePlanSettings();
  const qc = useQueryClient();
  const [draft, setDraft] = useState<PlanSettings | null>(null);
  const [apiKey, setApiKey] = useState("");
  const [calStatus, setCalStatus] = useState<{ ok: boolean; message: string } | null>(null);
  const [busy, setBusy] = useState(false);

  useEffect(() => {
    if (open && data) { setDraft(data); setApiKey(""); setCalStatus(null); }
  }, [open, data]);

  if (!draft) return null;
  const set = (patch: Partial<PlanSettings>) => setDraft({ ...draft, ...patch });
  const cloud = draft.aiKind === "openai" || draft.aiKind === "anthropic";

  const kindChanged = (kind: string) => {
    // Defaults depend on the provider; saving first would be surprising, so mirror the known ones.
    const defaults: Record<string, [string, string]> = { openai: ["https://api.openai.com/v1", "gpt-4o-mini"], anthropic: ["https://api.anthropic.com/v1", "claude-sonnet-5-5"], local: ["http://localhost:11434/v1", "llama3"] };
    const [defaultUrl, defaultModel] = defaults[kind] ?? ["", ""];
    set({ aiKind: kind, aiUrl: "", aiModel: "", defaultUrl, defaultModel, aiHasKey: false });
  };

  const save = async () => {
    setBusy(true);
    try {
      await api.savePlanSettings(draft);
      if (cloud && apiKey.trim()) await api.setAIKey(draft.aiKind, apiKey.trim());
      await qc.invalidateQueries({ queryKey: keys.planSettings });
      await qc.invalidateQueries({ queryKey: ["plan"] });
      toast.success("Planner settings saved");
      onOpenChange(false);
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Could not save the settings.");
    } finally {
      setBusy(false);
    }
  };

  const testCalendar = async () => {
    setBusy(true);
    try {
      setCalStatus(await api.testCalendar(draft.calendarUrl)); // tries what is typed; saves nothing
    } catch (err) {
      setCalStatus({ ok: false, message: err instanceof Error ? err.message : "Could not test the link." });
    } finally {
      setBusy(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg">
        <DialogHeader>
          <DialogTitle>Planner settings</DialogTitle>
          <DialogDescription>Working hours bound Auto-plan. The calendar and the AI are optional.</DialogDescription>
        </DialogHeader>

        <div className="flex flex-col gap-5">
          <fieldset className="flex items-end gap-3">
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="work-start">Work starts</Label>
              <Input id="work-start" type="time" value={toTimeInput(draft.workStart)} onChange={(e) => { const m = fromTimeInput(e.target.value); if (m !== null) set({ workStart: m }); }} />
            </div>
            <div className="flex flex-col gap-1.5">
              <Label htmlFor="work-end">Work ends</Label>
              <Input id="work-end" type="time" value={toTimeInput(draft.workEnd)} onChange={(e) => { const m = fromTimeInput(e.target.value); if (m !== null) set({ workEnd: m }); }} />
            </div>
          </fieldset>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="cal-url">Calendar link (iCal / ICS)</Label>
            <div className="flex gap-2">
              <Input id="cal-url" type="url" placeholder="https://calendar.google.com/…/basic.ics" value={draft.calendarUrl} onChange={(e) => { set({ calendarUrl: e.target.value }); setCalStatus(null); }} autoComplete="off" spellCheck={false} />
              <Button variant="outline" disabled={busy || !draft.calendarUrl.trim()} onClick={testCalendar}>Test</Button>
            </div>
            <p className="text-xs text-muted-foreground">
              Use your calendar's private "secret address in iCal format". Events show as locked time and Auto-plan avoids them. Zenith only reads the feed.
            </p>
            {calStatus && <p role="status" className={calStatus.ok ? "text-xs text-primary" : "text-xs text-destructive"}>{calStatus.message}</p>}
          </div>

          <div className="flex flex-col gap-1.5">
            <Label htmlFor="ai-kind">AI suggestions</Label>
            <select id="ai-kind" value={draft.aiKind} onChange={(e) => kindChanged(e.target.value)} className="h-8 rounded-lg border bg-background px-2 text-sm outline-none focus-visible:ring-2 focus-visible:ring-ring/50">
              {KINDS.map((k) => <option key={k.value} value={k.value}>{k.label}</option>)}
            </select>
            {draft.aiKind !== "" && (
              <div className="mt-2 flex flex-col gap-3">
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ai-url">Address</Label>
                  <Input id="ai-url" value={draft.aiUrl} placeholder={draft.defaultUrl} onChange={(e) => set({ aiUrl: e.target.value })} autoComplete="off" spellCheck={false} />
                </div>
                <div className="flex flex-col gap-1.5">
                  <Label htmlFor="ai-model">Model</Label>
                  <Input id="ai-model" value={draft.aiModel} placeholder={draft.defaultModel} onChange={(e) => set({ aiModel: e.target.value })} autoComplete="off" spellCheck={false} />
                </div>
                {cloud && (
                  <div className="flex flex-col gap-1.5">
                    <Label htmlFor="ai-key">API key</Label>
                    <Input id="ai-key" type="password" value={apiKey} placeholder={draft.aiHasKey ? "Saved in your keychain. Type to replace." : "Paste your API key"} onChange={(e) => setApiKey(e.target.value)} autoComplete="off" />
                    <p className="text-xs text-muted-foreground">
                      Stored in your system keychain. Asking for a plan sends the titles, estimates, priorities and deadlines of the tasks being placed, and anonymous busy times, to this provider. Notes, projects and calendar event titles are never sent.
                    </p>
                  </div>
                )}
                {!cloud && <p className="text-xs text-muted-foreground">Runs on this computer; nothing leaves it.</p>}
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="ghost" onClick={() => onOpenChange(false)}>Cancel</Button>
          <Button disabled={busy} onClick={save}>Save</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
