import { createSignal } from "solid-js";
import { System } from "~/mygo";

export type ThemeMode = "light" | "dark" | "system";

const STORAGE_KEY = "mtools-theme";

const systemDark = window.matchMedia("(prefers-color-scheme: dark)");

function loadMode(): ThemeMode {
  const v = localStorage.getItem(STORAGE_KEY);
  return v === "light" || v === "dark" || v === "system" ? v : "system";
}

function resolveDark(m: ThemeMode): boolean {
  return m === "system" ? systemDark.matches : m === "dark";
}

function applyTheme() {
  const dark = resolveDark(mode());
  setIsDark(dark);
  document.documentElement.classList.toggle("dark", dark);
}

// Keep the native window chrome in sync (background fill and macOS title
// bar buttons); the page colors themselves come from the CSS variables.
function syncWindow() {
  System.setWindowTheme(mode()).catch(() => {});
}

const [mode, setMode] = createSignal<ThemeMode>(loadMode());
const [isDark, setIsDark] = createSignal(resolveDark(mode()));

systemDark.addEventListener("change", applyTheme);
applyTheme();
syncWindow();

/** Current theme mode as stored ("system" follows the OS scheme). */
export const themeMode = mode;

/** Whether the resolved (applied) theme is dark. */
export const themeIsDark = isDark;

/** Set the theme mode; persists across launches. */
export function setThemeMode(m: ThemeMode) {
  setMode(m);
  localStorage.setItem(STORAGE_KEY, m);
  applyTheme();
  syncWindow();
}
