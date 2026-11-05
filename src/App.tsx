import { createResource, Show } from "solid-js";
import { System } from "./mygo";

export default function App() {
  const [info] = createResource(System.info);

  return (
    <div class="app">
      <header>
        <h1>M-Tools</h1>
        <p class="muted">个人工具箱</p>
      </header>
      <main>
        <div class="placeholder">
          <p>工具会出现在这里。</p>
          <p class="muted">第一个功能待规划…</p>
        </div>
      </main>
      <footer>
        <Show when={info()}>
          {(v) => (
            <span class="muted">
              {v().os}/{v().arch} · v{v().appVer} · {v().goVersion}
            </span>
          )}
        </Show>
      </footer>
    </div>
  );
}
