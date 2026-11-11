// Checks that every text/background pair in the design tokens meets WCAG AA
// (4.5:1) in both themes. Reads the tokens from src/index.css so the CSS stays
// the single source of truth. Run with `pnpm contrast`; it is part of `pnpm check`.
import { readFileSync } from "node:fs";

const css = readFileSync(new URL("../src/index.css", import.meta.url), "utf8");

function block(selector) {
  const m = css.match(new RegExp(`${selector}\\s*\\{([^}]*)\\}`));
  if (!m) throw new Error(`no ${selector} block in index.css`);
  const vars = {};
  for (const [, name, value] of m[1].matchAll(/--([\w-]+):\s*(#[0-9a-fA-F]{6})\s*;/g)) vars[name] = value;
  return vars;
}

const light = block(":root");
const dark = { ...light, ...block("\\.dark") };

const channel = (c) => {
  const s = c / 255;
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
};
const luminance = (hex) => {
  const n = parseInt(hex.slice(1), 16);
  return 0.2126 * channel((n >> 16) & 255) + 0.7152 * channel((n >> 8) & 255) + 0.0722 * channel(n & 255);
};
const ratio = (a, b) => {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x);
  return (hi + 0.05) / (lo + 0.05);
};

// [foreground, background] token pairs that carry text.
const pairs = [
  ["fg", "bg"], ["fg", "sidebar"], ["fg", "card"], ["fg", "selected"], ["fg", "hover"],
  ["muted", "bg"], ["muted", "sidebar"], ["muted", "card"], ["muted", "selected"], ["muted", "hover"],
  ["accent", "bg"], ["accent", "sidebar"], ["accent", "card"], ["accent", "selected"],
  ["danger", "bg"], ["danger", "sidebar"], ["danger", "card"], ["danger", "selected"],
  ["accent-fg", "accent"],
];

let failed = 0;
for (const [theme, tokens] of [["light", light], ["dark", dark]]) {
  for (const [fg, bg] of pairs) {
    const r = ratio(tokens[fg], tokens[bg]);
    const ok = r >= 4.5;
    if (!ok) failed++;
    console.log(`${ok ? "ok  " : "FAIL"} ${theme.padEnd(5)} ${fg} on ${bg}: ${r.toFixed(2)}:1`);
  }
}
if (failed) {
  console.error(`\n${failed} pair(s) below 4.5:1`);
  process.exit(1);
}
console.log("\nAll text pairs meet 4.5:1.");
