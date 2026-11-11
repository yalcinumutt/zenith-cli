import js from "@eslint/js";
import globals from "globals";
import reactHooks from "eslint-plugin-react-hooks";
import tseslint from "typescript-eslint";

export default tseslint.config(
  { ignores: ["dist", "bindings"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["src/**/*.{ts,tsx}"],
    languageOptions: { globals: globals.browser },
    plugins: { "react-hooks": reactHooks },
    rules: {
      "react-hooks/rules-of-hooks": "error",
      "react-hooks/exhaustive-deps": "warn",
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_" }],
      // The generated bindings may only be imported by the client module (ADR 0001, 0007).
      "no-restricted-imports": ["error", { patterns: [{ group: ["**/bindings/**"], message: "Import from src/api/client.ts instead." }] }],
    },
  },
  {
    files: ["src/api/client.ts"],
    rules: { "no-restricted-imports": "off" },
  },
);
