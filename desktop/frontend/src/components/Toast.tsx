import { useEffect } from "react";
import { useUI } from "../state/ui";

/** A transient, polite status message (for example a refused completion). */
export function Toast() {
  const toast = useUI((s) => s.toast);
  useEffect(() => {
    if (!toast) return;
    const t = setTimeout(() => useUI.setState({ toast: null }), 5000);
    return () => clearTimeout(t);
  }, [toast]);
  return (
    <div aria-live="polite" role="status" className="pointer-events-none fixed inset-x-0 top-4 z-50 flex justify-center px-4">
      {toast && (
        <div className="pop-in pointer-events-auto max-w-md rounded-lg bg-fg px-4 py-2 text-sm text-bg shadow-lg">{toast.text}</div>
      )}
    </div>
  );
}
