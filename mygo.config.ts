import { defineConfig } from "mygo-cli";

export default defineConfig({
  name: "m-tools",
  identifier: "com.sdttttt.mtools",
  version: "2026.11.05",
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
  // Ship the third-party license notices next to the executable
  // (Contents/Resources on macOS): MIT/Apache-2.0 require the notice to
  // accompany distributed copies.
  resources: ["THIRD-PARTY-NOTICES.md"],
  out: "build",
});
