import { defineConfig, presetWind3 } from "unocss";

export default defineConfig({
  presets: [presetWind3()],
  // Follow the system light/dark scheme, matching the window's
  // light-dark() BackgroundColor in main.go.
  dark: "media",
  theme: {
    colors: {
      // Semantic tokens backed by the CSS variables defined in the
      // preflight below, which flip between light/dark via media query.
      bg: "var(--bg)",
      fg: "var(--fg)",
      muted: "var(--muted)",
      card: "var(--card)",
      line: "var(--border)",
      accent: "var(--accent)",
    },
  },
  // Base tokens and resets live here — the project contains no hand-written
  // .css files; all styling is UnoCSS utilities plus this preflight.
  preflights: [
    {
      getCSS: () => `
        :root {
          color-scheme: light dark;
          --bg: #f6f7f9;
          --fg: #1a1d21;
          --muted: #6b7280;
          --card: #ffffff;
          --border: #e5e7eb;
          --accent: #4f6bed;
        }
        @media (prefers-color-scheme: dark) {
          :root {
            --bg: #0f1115;
            --fg: #e6e8eb;
            --muted: #9ca3af;
            --card: #171a20;
            --border: #262b33;
            --accent: #7b96f5;
          }
        }
        * { box-sizing: border-box; }
        body {
          margin: 0;
          font-family: system-ui, -apple-system, "Segoe UI", "PingFang SC", "Microsoft YaHei", sans-serif;
          background: var(--bg);
          color: var(--fg);
        }
      `,
    },
  ],
});
