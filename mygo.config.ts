import { defineConfig } from "mygo-cli";

export default defineConfig({
  name: "m-tools",
  identifier: "com.sdttttt.mtools",
  version: "0.1.0",
  // Go entrypoint: cmd/m-tools (Go standard layout). The ./ prefix is
  // required — bare "cmd/m-tools" collides with the std library's cmd/ tree.
  main: "./cmd/m-tools",
  // `mygo dev` runs devCommand and loads devUrl; `mygo build` runs
  // buildCommand and embeds frontendDist into the app.
  devUrl: "http://localhost:5173",
  devCommand: "cd webview && bun run dev",
  buildCommand: "cd webview && bun run build",
  frontendDist: "webview/dist",
  bindings: "webview/src/mygo.ts",
  out: "build",
});
