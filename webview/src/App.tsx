import { createResource, createSignal, For, Show } from "solid-js";
import { System } from "./mygo";
import Mp4GifTool from "./tools/Mp4Gif";

type ToolId = "mp4gif";

const TOOLS: { id: ToolId; name: string; desc: string }[] = [
  { id: "mp4gif", name: "MP4 → GIF", desc: "视频转 GIF 动图，可裁剪、调帧率与画质" },
];

export default function App() {
  const [tool, setTool] = createSignal<ToolId | null>(null);
  const [info] = createResource(System.info);

  return (
    <div class="flex flex-col min-h-screen">
      <header class="flex items-center gap-3 px-8 pt-6 pb-2">
        <Show when={tool()} fallback={<h1 class="m-0 text-xl font-semibold">M-Tools</h1>}>
          {(t) => (
            <>
              <button
                class="cursor-pointer rounded border border-line bg-card px-2 py-1 text-sm hover:border-accent"
                onClick={() => setTool(null)}
              >
                ← 返回
              </button>
              <h1 class="m-0 text-xl font-semibold">{TOOLS.find((x) => x.id === t())?.name}</h1>
            </>
          )}
        </Show>
        <p class="m-0 mt-1 text-sm text-muted">个人工具箱</p>
      </header>
      <main class="flex-1 px-8 py-4">
        <Show
          when={tool()}
          fallback={
            <div class="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-4">
              <For each={TOOLS}>
                {(t) => (
                  <button
                    class="cursor-pointer rounded-xl border border-line bg-card p-4 text-left hover:border-accent"
                    onClick={() => setTool(t.id)}
                  >
                    <p class="m-0 text-base font-medium">{t.name}</p>
                    <p class="m-0 mt-1 text-xs text-muted">{t.desc}</p>
                  </button>
                )}
              </For>
            </div>
          }
        >
          {tool() === "mp4gif" && <Mp4GifTool />}
        </Show>
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
