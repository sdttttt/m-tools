import { createSignal, For, Show } from "solid-js";
import { ImageConv, type ImgconvConvertResult, type Info as ImageInfo } from "~/mygo";
import { Button } from "~/components/ui/button";
import { Card, CardContent } from "~/components/ui/card";
import { Label } from "~/components/ui/label";
import { InfoTip } from "~/components/InfoTip";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/components/ui/select";
import {
  FORMAT_LABELS,
  LABEL_BY_QUALITY,
  QUALITY_BY_LABEL,
  QUALITY_LABELS,
  STATE_COLORS,
  STATE_LABELS,
  fmtSize,
  fileName,
  probeAll,
  sourceFormat,
  type ImgFormat,
  type RowState,
} from "~/lib/img";

interface Row {
  info: ImageInfo;
  state: RowState;
  result: ImgconvConvertResult | null;
  error: string;
}

export default function ImageConvTool() {
  const [supported, setSupported] = createSignal(true);
  const [rows, setRows] = createSignal<Row[]>([]);
  const [target, setTarget] = createSignal<ImgFormat>("png");
  const [quality, setQuality] = createSignal<"high" | "balanced" | "compact">("balanced");
  const [running, setRunning] = createSignal(false);
  const [notice, setNotice] = createSignal("");
  let stopRequested = false;

  ImageConv.supported()
    .then(setSupported)
    .catch(() => setSupported(false));

  // Options/value/onChange all live in label-land: Kobalte matches the
  // trigger text against the options array, so handing it ids while
  // displaying labels breaks the trigger (it would show nothing).
  const targetOptions = () =>
    Object.entries(FORMAT_LABELS)
      .filter(([id]) => id !== "webp" || supported())
      .map(([, label]) => label);

  const LABEL_BY_FORMAT: Record<string, string> = Object.fromEntries(
    Object.entries(FORMAT_LABELS).map(([id, label]) => [label, id]),
  );

  function updateRow(path: string, patch: (row: Row) => Row) {
    setRows((prev) => prev.map((r) => (r.info.path === path ? patch(r) : r)));
  }

  async function pick() {
    if (running()) return;
    setNotice("");
    const paths = await ImageConv.pickImages();
    if (!paths || paths.length === 0) return;
    const probed = await probeAll(ImageConv.probe, paths);
    setRows(
      probed.map((info) => {
        const same = sourceFormat(info.path, info.format) === target() && info.supported;
        return {
          info,
          state: same || !info.supported ? "skipped" : "pending",
          result: null,
          error: "",
        };
      }),
    );
  }

  function patchRowForTarget() {
    // A target change re-evaluates which rows would be same-format skips.
    setRows((prev) =>
      prev.map((r) => {
        if (r.state === "converting" || r.state === "done") return r;
        const same = sourceFormat(r.info.path, r.info.format) === target() && r.info.supported;
        return {
          ...r,
          state: r.info.supported && !same ? "pending" : "skipped",
          error: "",
        };
      }),
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
      updateRow(r.info.path, (row) => ({ ...row, state: "converting" }));
      try {
        const res = await ImageConv.convert({
          path: r.info.path,
          targetFormat: target(),
          quality: quality(),
        });
        updateRow(r.info.path, (row) => ({ ...row, state: "done", result: res }));
        anyDone = true;
      } catch (e) {
        updateRow(r.info.path, (row) => ({
          ...row,
          state: "failed",
          error: String(e).replace(/^Error:\s*/, ""),
        }));
      }
    }
    setRunning(false);
    if (stopRequested) setNotice("已取消，剩余图片保持待转换");
    else if (anyDone) setNotice("全部完成，输出保存在原图所在文件夹");
  }

  async function save(row: Row) {
    if (!row.result) return;
    const dest = await ImageConv.saveAs(row.result.outputPath, fileName(row.result.outputPath));
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
              : "点击选择图片（PNG / JPEG / GIF / WebP / BMP / TIFF，可多选）"}
          </p>
          <p class="m-0 text-xs text-muted-foreground">
            {supported()
              ? "转换保持原尺寸原画质规格，输出保存在原图旁；调整尺寸请用「图片大小」工具"
              : "当前构建不含 WebP 编解码（需 -tags with_webp 构建）"}
          </p>
        </CardContent>
      </Card>

      {/* 参数 */}
      <Show when={rows().length > 0}>
        <Card>
          <CardContent class="flex flex-col gap-4 px-5 py-4">
            <div class="flex flex-wrap items-end gap-x-6 gap-y-3">
              <div class="flex flex-col gap-1">
                <Label class="text-xs text-muted-foreground">目标格式</Label>
                <Select
                  options={targetOptions()}
                  value={FORMAT_LABELS[target()]}
                  onChange={(label) => {
                    if (!label) return;
                    setTarget((LABEL_BY_FORMAT[label] ?? target()) as ImgFormat);
                    patchRowForTarget();
                  }}
                  itemComponent={(props) => (
                    <SelectItem item={props.item}>{props.item.rawValue}</SelectItem>
                  )}
                >
                  <SelectTrigger class="w-40" aria-label="目标格式">
                    <SelectValue<string>>{(state) => state.selectedOption()}</SelectValue>
                  </SelectTrigger>
                  <SelectContent />
                </Select>
              </div>
              <div class="flex flex-col gap-1">
                <div class="flex items-center gap-1">
                  <Label class="text-xs text-muted-foreground">画质</Label>
                  <InfoTip label="画质说明">
                    画质仅影响 JPEG / WebP 这类有损格式；PNG / GIF / BMP / TIFF
                    是无损格式，画质框不可用。高画质≈92、均衡≈82、紧凑≈62（0-100）
                  </InfoTip>
                </div>
                <Select
                  options={[...QUALITY_LABELS]}
                  value={LABEL_BY_QUALITY[quality()]}
                  disabled={target() !== "jpg" && target() !== "webp"}
                  onChange={(label) => {
                    if (label) setQuality(QUALITY_BY_LABEL[label] ?? "balanced");
                  }}
                  itemComponent={(props) => (
                    <SelectItem item={props.item}>{props.item.rawValue}</SelectItem>
                  )}
                >
                  <SelectTrigger class="w-40" aria-label="画质">
                    <SelectValue<string>>{(state) => state.selectedOption()}</SelectValue>
                  </SelectTrigger>
                  <SelectContent />
                </Select>
              </div>
            </div>

            {/* 任务列表 */}
            <div class="flex flex-col gap-2 border-t pt-3">
              <For each={rows()}>
                {(r) => (
                  <div class="flex flex-col gap-1 rounded-md border px-3 py-2">
                    <div class="flex items-center justify-between gap-3">
                      <div class="min-w-0">
                        <p class="m-0 truncate text-sm">
                          {fileName(r.info.path)}
                          {r.info.supported && (
                            <span class="ml-2 text-xs text-muted-foreground">
                              {r.info.width}×{r.info.height} · {fmtSize(r.info.sizeBytes)}
                              {r.info.hasAlpha ? " · 含透明通道" : ""}
                            </span>
                          )}
                        </p>
                        <Show
                          when={r.state === "failed" || r.state === "skipped"}
                          fallback={
                            <Show when={r.state === "done" && r.result}>
                              {(res) => (
                                <p class="m-0 text-xs text-muted-foreground">
                                  → {res().format.toUpperCase()} · {res().width}×{res().height} ·{" "}
                                  {fmtSize(res().sizeBytes)} ·{" "}
                                  {(res().durationMs / 1000).toFixed(1)} 秒
                                </p>
                              )}
                            </Show>
                          }
                        >
                          <p class="m-0 text-xs text-destructive">
                            {r.state === "skipped"
                              ? r.info.skipReason ||
                                `与目标格式相同（${sourceFormat(r.info.path, r.info.format).toUpperCase()}）`
                              : r.error}
                          </p>
                        </Show>
                      </div>
                      <div class="flex shrink-0 items-center gap-2">
                        <span class={`text-xs ${STATE_COLORS[r.state]}`}>
                          {STATE_LABELS[r.state]}
                        </span>
                        <Show when={r.state === "done" && r.result}>
                          <Button variant="outline" size="sm" onClick={() => save(r)}>
                            保存到…
                          </Button>
                          <Button
                            variant="outline"
                            size="sm"
                            onClick={() => ImageConv.openFolder(r.result!.outputPath)}
                          >
                            打开文件夹
                          </Button>
                        </Show>
                      </div>
                    </div>
                    <Show when={r.state === "done" && r.result?.previewDataURL}>
                      <img
                        class="max-h-40 self-start rounded-md"
                        src={r.result!.previewDataURL}
                        alt="转换预览"
                      />
                    </Show>
                  </div>
                )}
              </For>
            </div>

            {/* 操作 */}
            <div class="flex items-center gap-3">
              <Show
                when={!running()}
                fallback={
                  <Button
                    variant="outline"
                    onClick={() => {
                      stopRequested = true;
                      ImageConv.cancelActive();
                    }}
                  >
                    取消
                  </Button>
                }
              >
                <Button onClick={run} disabled={!hasPending()}>
                  开始转换
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
