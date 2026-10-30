// ESLint flat config used by super-linter (LINTER_RULES_PATH=.github/linters).
// The only JavaScript in this repo is the GNOME Shell extensions, which run
// under GJS as legacy (non-module) scripts.
//
// - sourceType "script": GNOME Shell calls the top-level init(), enable(),
//   disable() and buildPrefsWidget() functions itself. Each file lists them
//   in an /* exported ... */ comment, which ESLint only honours for scripts.
// - GJS runtime globals (imports, log, _ ...) are declared per file with
//   /* global ... */ comments so Codacy's ESLint, which does not read this
//   file, sees the same declarations.
import js from "@eslint/js";
import globals from "globals";

export default [
  js.configs.recommended,
  {
    files: ["gnome-shell-extension/**/*.js", "gnome-extension/**/*.js"],
    languageOptions: {
      ecmaVersion: "latest",
      sourceType: "script",
      globals: { ...globals.es2021 },
    },
  },
];
