import { defineConfig } from "vite";
import solid from "vite-plugin-solid";
import { fileURLToPath } from "node:url";
import path from "node:path";
import tailwindcss from "tailwindcss";
import autoprefixer from "autoprefixer";

const rootDir = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [solid()],
  resolve: {
    // solid-ui components import via the ~/ alias (see ui.config.json).
    alias: { "~": path.resolve(rootDir, "src") },
  },
  // PostCSS runs inline (Tailwind + autoprefixer) so no separate
  // postcss.config.* file is needed; Tailwind resolves tailwind.config.ts.
  css: {
    postcss: { plugins: [tailwindcss(), autoprefixer()] },
  },
  server: {
    // The app loads this address during `mygo dev` (devUrl in mygo.config.ts).
    port: 5173,
    strictPort: true,
    // The development app and the packaged builds are not the page's.
    watch: { ignored: ["**/.mygo/**", "**/build/**"] },
  },
});
