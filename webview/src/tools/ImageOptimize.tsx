import { createSignal, For, Show } from "solid-js";
import { ImageOptimize, type Info as ImageInfo, type OptimizeResult } from "~/mygo";
import { Button } from "~/components/ui/button";
import { Card, CardContent } from "~/components/ui/card";
import { InfoTip } from "~/components/InfoTip";
import { STATE_COLORS, STATE_LABELS, fmtSize, fileName, probeAll, type RowState } from "~/lib/img";

interface Row {
  info: ImageInfo;
  state: RowState;
  result: OptimizeResult | null;
  error: string;
}

export default function ImageOptimizeTool() {
  const [rows, setRows] = createSignal<Row[]>([]);
  const [running, setRunning] = createSignal(false);
  const [notice, setNotice] = createSignal("");
  let stopRequested = false;

  function updateRow(path: string, patch: (row: Row) => Row) {
    setRows((prev) => prev.map((r) => (r.info.path === path ? patch(r) : r)));
  }

  async function pick() {
    if (running()) return;
    setNotice("");
    const paths = await ImageOptimize.pickImages();
    if (!paths || paths.length === 0) return;
    const probed = await probeAll(ImageOptimize.probe, paths);
    setRows(
      probed.map((info) => ({
        info,
        state: info.supported ? "pending" : "skipped",
        result: null,
        error: "",
      })),
    );
  }

  async function run() {
    if (running()) return;
    setRunning(true);
    setNotice("");
    stopRequested = false;
    let anyDone = false;
    for (const r of rows()) {
      if (stopRequested) break;
      if (r.state !== "pending") continue;
      updateRow(r.info.path, (row) => ({ ...row, state: "optimizing" }));
      try {
        const res = await ImageOptimize.optimize({ path: r.info.path });
        if (res.skipReason) {
          updateRow(r.info.path, (row) => ({
            ...row,
            state: "skipped",
            error: res.skipReason,
          }));
        } else if (res.notNeeded) {
          updateRow(r.info.path, (row) => ({ ...row, state: "done", result: res }));
        } else {
          updateRow(r.info.path, (row) => ({ ...row, state: "done", result: res }));
          anyDone = true;
        }
      } catch (e) {
        updateRow(r.info.path, (row) => ({
          ...row,
          state: "failed",
          error: String(e).replace(/^Error:\s*/, ""),
        }));
      }
    }
    setRunning(false);
    if (stopRequested) setNotice("已取消，剩余图片保持待处理");
    else if (anyDone) setNotice("全部完成，优化结果以 -min 副本保存在原图旁");
  }

  async function save(row: Row) {
    if (!row.result?.outputPath) return;
    const dest = await ImageOptimize.saveAs(row.result.outputPath, fileName(row.result.outputPath));
    if (dest) setNotice(`已另存到 ${dest}`);
  }

  const hasPending = () => rows().some((r) => r.state === "pending");

  return (
    <div class="mx-auto flex w-full max-w-2xl flex-col gap-5">
      {/* 文件选择 */}
      <Card
        class="cursor-pointer border-dashed transition-colors hover:border-primary"
        onClick={() => !running() && pick()}
      >
        <CardContent class="flex flex-col gap-1 px-6 py-5">
          <p class="m-0 text-sm">
            {rows().length > 0
              ? `已选择 ${rows().length} 张图片，点击可重新选择`
              : "点击选择图片（PNG / JPEG / TIFF / GIF / WebP，可多选）"}
          </p>
          <p class="m-0 text-xs text-muted-foreground">
            像素级无损压缩：精度和尺寸不变，结果以 -min 副本保存在原图旁，不会更大
            <span class="ml-2 inline-flex items-center gap-0.5">
              动图与 BMP 自动跳过
              <InfoTip label="优化说明">
                PNG 走最高压缩并尝试索引色；JPEG 只剥离 EXIF 等元数据，绝不重新编码；TIFF 转 Deflate
                压缩；静态 GIF 重编码；WebP 需 with_webp
                构建并转无损格式。压缩后若没有变小，不会写出文件（标「已是最优」）。
              </InfoTip>
            </span>
          </p>
        </CardContent>
      </Card>

      {/* 任务列表 */}
      <Show when={rows().length > 0}>
        <Card>
          <CardContent class="flex flex-col gap-2 px-5 py-4">
            <For each={rows()}>
              {(r) => (
                <div class="flex items-center justify-between gap-3 rounded-md border px-3 py-2">
                  <div class="min-w-0">
                    <p class="m-0 truncate text-sm">
                      {fileName(r.info.path)}
                      {r.info.supported && (
                        <span class="ml-2 text-xs text-muted-foreground">
                          {r.info.format.toUpperCase()} · {r.info.width}×{r.info.height} ·{" "}
                          {fmtSize(r.info.sizeBytes)}
                        </span>
                      )}
                    </p>
                    <Show
                      when={r.state !== "done" || !r.result}
                      fallback={
                        <Show
                          when={r.result?.notNeeded}
                          fallback={
                            <p class="m-0 text-xs text-primary">
                              → {fmtSize(r.result!.originalBytes)} →{" "}
                              {fmtSize(r.result!.optimizedBytes)}（省
                              {Math.max(
                                1,
                                Math.round(
                                  (1 - r.result!.optimizedBytes / r.result!.originalBytes) * 100,
                                ),
                              )}
                              %）
                            </p>
                          }
                        >
                          <p class="m-0 text-xs text-muted-foreground">已是最优，无需优化</p>
                        </Show>
                      }
                    >
                      <Show when={r.state === "failed" || r.state === "skipped"}>
                        <p class="m-0 text-xs text-destructive">
                          {r.state === "skipped" ? r.info.skipReason || r.error : r.error}
                        </p>
                      </Show>
                    </Show>
                  </div>
                  <div class="flex shrink-0 items-center gap-2">
                    <span class={`text-xs ${STATE_COLORS[r.state]}`}>{STATE_LABELS[r.state]}</span>
                    <Show when={r.state === "done" && r.result?.outputPath}>
                      <Button variant="outline" size="sm" onClick={() => save(r)}>
                        保存到…
                      </Button>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => ImageOptimize.openFolder(r.result!.outputPath)}
                      >
                        打开文件夹
                      </Button>
                    </Show>
                  </div>
                </div>
              )}
            </For>

            {/* 操作 */}
            <div class="flex items-center gap-3 border-t pt-3">
              <Show
                when={!running()}
                fallback={
                  <Button
                    variant="outline"
                    onClick={() => {
                      stopRequested = true;
                      ImageOptimize.cancelActive();
                    }}
                  >
                    取消
                  </Button>
                }
              >
                <Button onClick={run} disabled={!hasPending()}>
                  开始优化
                </Button>
              </Show>
              <Show when={notice()}>
                <span class="truncate text-sm text-muted-foreground">{notice()}</span>
              </Show>
            </div>
          </CardContent>
        </Card>
      </Show>
    </div>
  );
}
