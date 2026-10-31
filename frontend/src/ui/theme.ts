/** Theme and editor font: what the app looks like, independent of any component. */
import { derived, readable, writable } from "svelte/store";

export type Theme = "system" | "light" | "dark";

const CYCLE: Record<Theme, Theme> = { system: "light", light: "dark", dark: "system" };
const ICONS: Record<Theme, string> = { system: "◐", light: "☀", dark: "☾" };

/** "system" follows the OS, "light" and "dark" pin it. Persisted as settings.theme. */
export const theme = writable<Theme>("system");

const systemDark = readable(false, (set) => {
  const mq = window.matchMedia("(prefers-color-scheme: dark)");
  set(mq.matches);
  const onChange = (e: MediaQueryListEvent) => set(e.matches);
  mq.addEventListener("change", onChange);
  return () => mq.removeEventListener("change", onChange);
});

export const effectiveTheme = derived([theme, systemDark], ([$theme, $dark]) =>
  $theme === "system" ? ($dark ? "dark" : "light") : $theme,
);

export const nextTheme = (t: Theme): Theme => CYCLE[t];
export const themeIcon = (t: Theme): string => ICONS[t];

export function isTheme(v: unknown): v is Theme {
  return v === "system" || v === "light" || v === "dark";
}

/** Set the editor's font through the CSS variables Editor.svelte reads; bad values fall back. */
export function applyEditorFont(font: { family?: string; size?: number }): void {
  const root = document.documentElement;
  if (font.family?.trim()) root.style.setProperty("--editor-font-family", font.family);
  else root.style.removeProperty("--editor-font-family");
  if (typeof font.size === "number" && font.size >= 8 && font.size <= 48) {
    root.style.setProperty("--editor-font-size", `${font.size}px`);
  } else {
    root.style.removeProperty("--editor-font-size");
  }
}
