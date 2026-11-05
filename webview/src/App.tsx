import { createResource, createSignal, For, Show } from "solid-js";
import { System } from "~/mygo";
import { Button } from "~/components/ui/button";
import { Card, CardContent, CardDescription, CardTitle } from "~/components/ui/card";
import { ThemeToggle } from "~/components/ThemeToggle";
import Mp4GifTool from "~/tools/Mp4Gif";
import ImageConvTool from "~/tools/ImageConv";

type ToolId = "mp4gif" | "imgconv";

const TOOLS: { id: ToolId; name: string; desc: string }[] = [
  { id: "mp4gif", name: "MP4 → GIF", desc: "视频转 GIF 动图，可裁剪、调帧率与画质" },
  {
    id: "imgconv",
    name: "图片格式转换",
    desc: "PNG / JPEG / GIF / WebP / BMP / TIFF 互转，支持批量",
  },
];

export default function App() {
  const [tool, setTool] = createSignal<ToolId | null>(null);
  const [info] = createResource(System.info);

  return (
    <div class="flex min-h-screen flex-col">
      <header class="flex items-center justify-between gap-3 px-8 pb-2 pt-6">
        <div class="flex items-center gap-3">
          <Show when={tool()} fallback={<h1 class="m-0 text-xl font-semibold">M-Tools</h1>}>
            {(t) => (
              <>
                <Button variant="outline" size="sm" onClick={() => setTool(null)}>
                  ← 返回
                </Button>
                <h1 class="m-0 text-xl font-semibold">{TOOLS.find((x) => x.id === t())?.name}</h1>
              </>
            )}
          </Show>
          <p class="m-0 mt-1 text-sm text-muted-foreground">个人工具箱</p>
        </div>
        <ThemeToggle />
      </header>
      <main class="flex-1 px-8 py-4">
        <Show
          when={tool()}
          fallback={
            <div class="grid grid-cols-[repeat(auto-fill,minmax(220px,1fr))] gap-4">
              <For each={TOOLS}>
                {(t) => (
                  <Card
                    class="cursor-pointer transition-colors hover:border-primary"
                    onClick={() => setTool(t.id)}
                  >
                    <CardContent class="p-4">
                      <CardTitle class="text-base font-medium">{t.name}</CardTitle>
                      <CardDescription class="mt-1 text-xs">{t.desc}</CardDescription>
                    </CardContent>
                  </Card>
                )}
              </For>
            </div>
          }
        >
          {tool() === "mp4gif" && <Mp4GifTool />}
          {tool() === "imgconv" && <ImageConvTool />}
        </Show>
      </main>
      <footer class="px-8 py-3 text-xs">
        <Show when={info()}>
          {(v) => (
            <span class="text-muted-foreground">
              {v().os}/{v().arch} · v{v().appVer} · {v().goVersion}
            </span>
          )}
        </Show>
      </footer>
    </div>
  );
}
