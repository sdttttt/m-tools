import { defineConfig } from "vite";
import unocss from "unocss/vite";
import solid from "vite-plugin-solid";

export default defineConfig({
  plugins: [unocss(), solid()],
  server: {
    // The app loads this address during `mygo dev` (devUrl in mygo.config.ts).
    port: 5173,
    strictPort: true,
    // The development app and the packaged builds are not the page's.
    watch: { ignored: ["**/.mygo/**", "**/build/**"] },
  },
});
