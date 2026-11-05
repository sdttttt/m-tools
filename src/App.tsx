import { createResource, Show } from "solid-js";
import { System } from "./mygo";

export default function App() {
  const [info] = createResource(System.info);

  return (
    <div class="flex flex-col min-h-screen">
      <header class="px-8 pt-6 pb-2">
        <h1 class="m-0 text-xl font-semibold">M-Tools</h1>
        <p class="mt-1 text-sm text-muted">个人工具箱</p>
      </header>
      <main class="flex-1 grid place-items-center px-8 py-4">
        <div class="border-line rounded-xl border border-dashed px-16 py-12 text-center">
          <p>工具会出现在这里。</p>
          <p class="mt-1 text-sm text-muted">第一个功能待规划…</p>
        </div>
      </main>
      <footer class="px-8 py-3 text-xs">
        <Show when={info()}>
          {(v) => (
            <span class="text-muted">
              {v().os}/{v().arch} · v{v().appVer} · {v().goVersion}
            </span>
          )}
        </Show>
      </footer>
    </div>
  );
}
