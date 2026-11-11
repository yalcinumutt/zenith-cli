export const isMac: boolean =
  typeof navigator !== "undefined" && /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);

/** Mod is Command on macOS and Control everywhere else. */
export const modSymbol = isMac ? "⌘" : "Ctrl";

export function formatShortcut(shortcut: string): string {
  return shortcut
    .split("+")
    .map((p) => {
      switch (p) {
        case "mod": return modSymbol;
        case "shift": return isMac ? "⇧" : "Shift";
        case "alt": return isMac ? "⌥" : "Alt";
        case "arrowup": return "↑";
        case "arrowdown": return "↓";
        case "backspace": return "⌫";
        case "space": return "Space";
        case "escape": return "Esc";
        case "enter": return "↵";
        default: return p.length === 1 ? p.toUpperCase() : p;
      }
    })
    .join(isMac ? "" : "+");
}

/** True when the key event matches a shortcut such as "mod+k", "x" or "mod+arrowup". */
export function matches(e: KeyboardEvent, shortcut: string): boolean {
  const parts = shortcut.toLowerCase().split("+");
  const key = parts[parts.length - 1];
  const wantMod = parts.includes("mod");
  const wantShift = parts.includes("shift");
  const wantAlt = parts.includes("alt");
  const mod = isMac ? e.metaKey : e.ctrlKey;
  if (wantMod !== mod) return false;
  if (e.shiftKey !== wantShift || e.altKey !== wantAlt) return false;
  // Don't treat the other platform's modifier as a plain key.
  if (!wantMod && (e.metaKey || e.ctrlKey)) return false;
  const k = e.key === " " ? "space" : e.key.toLowerCase();
  return k === key;
}

export function isTyping(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null;
  if (!el) return false;
  const tag = el.tagName;
  return tag === "INPUT" || tag === "TEXTAREA" || tag === "SELECT" || el.isContentEditable;
}
