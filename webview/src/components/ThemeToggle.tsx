import { ToggleGroup, ToggleGroupItem } from "~/components/ui/toggle-group";
import { setThemeMode, themeMode, type ThemeMode } from "~/lib/theme";

const MODES: { value: ThemeMode; label: string }[] = [
  { value: "system", label: "跟随系统" },
  { value: "light", label: "浅色" },
  { value: "dark", label: "深色" },
];

/** Switches between following the OS scheme and forcing light/dark. */
export function ThemeToggle() {
  return (
    <ToggleGroup
      variant="outline"
      value={themeMode()}
      onChange={(value) => {
        if (value === "light" || value === "dark" || value === "system") {
          setThemeMode(value);
        }
      }}
    >
      {MODES.map((m) => (
        <ToggleGroupItem value={m.value} aria-label={m.label}>
          {m.label}
        </ToggleGroupItem>
      ))}
    </ToggleGroup>
  );
}
