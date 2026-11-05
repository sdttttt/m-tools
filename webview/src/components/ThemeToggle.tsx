import type { JSX } from "solid-js";
import { Tooltip, TooltipContent, TooltipTrigger } from "~/components/ui/tooltip";
import { Button } from "~/components/ui/button";
import { setThemeMode, themeMode, type ThemeMode } from "~/lib/theme";

// Inline lucide-style icons (stroke inherits currentColor), matching the
// InfoTip icon approach — no icon library in this project.
function Icon(props: { children: JSX.Element }) {
  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      class="size-4"
    >
      {props.children}
    </svg>
  );
}

const SunIcon = () => (
  <Icon>
    <circle cx="12" cy="12" r="4" />
    <path d="M12 2v2" />
    <path d="M12 20v2" />
    <path d="m4.93 4.93 1.41 1.41" />
    <path d="m17.66 17.66 1.41 1.41" />
    <path d="M2 12h2" />
    <path d="M20 12h2" />
    <path d="m6.34 17.66-1.41 1.41" />
    <path d="m19.07 4.93-1.41 1.41" />
  </Icon>
);

const MoonIcon = () => (
  <Icon>
    <path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z" />
  </Icon>
);

const MonitorIcon = () => (
  <Icon>
    <rect width="20" height="14" x="2" y="3" rx="2" />
    <line x1="8" x2="16" y1="21" y2="21" />
    <line x1="12" x2="12" y1="17" y2="21" />
  </Icon>
);

// Cycle order: click advances light -> dark -> system -> light.
const NEXT: Record<ThemeMode, ThemeMode> = {
  light: "dark",
  dark: "system",
  system: "light",
};

const ICONS: Record<ThemeMode, () => JSX.Element> = {
  light: SunIcon,
  dark: MoonIcon,
  system: MonitorIcon,
};

const LABELS: Record<ThemeMode, string> = {
  light: "浅色",
  dark: "深色",
  system: "跟随系统",
};

/** One button that cycles light -> dark -> system, showing the current mode. */
export function ThemeToggle() {
  const next = () => setThemeMode(NEXT[themeMode()]);
  return (
    <Tooltip placement="bottom">
      <TooltipTrigger
        as={Button}
        variant="outline"
        size="icon"
        onClick={next}
        aria-label="切换主题"
      >
        {ICONS[themeMode()]()}
      </TooltipTrigger>
      <TooltipContent class="text-xs">
        {LABELS[themeMode()]}（点击切换到{LABELS[NEXT[themeMode()]]}）
      </TooltipContent>
    </Tooltip>
  );
}
