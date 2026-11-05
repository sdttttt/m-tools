import { createSignal, For, Show } from "solid-js";
import { ImageResize, type Info as ImageInfo, type ResizeResult } from "~/mygo";
import { MotionButton } from "~/components/motion";
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
import { TextField, TextFieldInput } from "~/components/ui/text-field";
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
  type ProbeRow,
  type RowState,
} from "~/lib/img";

interface Row {
  info: ImageInfo;
  state: RowState;
  result: ResizeResult | null;
  error: string;
}

type Mode = "percent" | "exact";

// Output format defaults to the first picked image's own format, so a pure
// resize keeps the format unless the user picks another one. Options stay
// bare labels: Kobalte matches `value` against options verbatim, so any
// suffix (e.g. a "（当前）" marker) would blank the trigger text.
function formatOptions(supported: boolean): string[] {
  return Object.entries(FORMAT_LABELS)
    .filter(([id]) => id !== "webp" || supported)
    .map(([, label]) => label);
}

export default function ImageResizeTool() {
  const [supported, setSupported] = createSignal(true);
  const [rows, setRows] = createSignal<Row[]>([]);
  const [format, setFormat] = createSignal<ImgFormat>("png");
  const [quality, setQuality] = createSignal<"high" | "balanced" | "compact">("balanced");
  const [mode, setMode] = createSignal<Mode>("percent");
  const [percent, setPercent] = createSignal(100);
  const [width, setWidth] = createSignal(0);
  const [height, setHeight] = createSignal(0);
  const [lock, setLock] = createSignal(true);
  const [running, setRunning] = createSignal(false);
  const [notice, setNotice] = createSignal("");
  let stopRequested = false;

  ImageResize.supported()
    .then(setSupported)
    .catch(() => setSupported(false));

  const LABEL_BY_FORMAT: Record<string, string> = Object.fromEntries(
    Object.entries(FORMAT_LABELS).map(([id, label]) => [label, id]),
  );

  const firstSupported = (): ProbeRow | undefined =>
    rows().find((r) => r.info.supported && r.info.width > 0)?.info;

  // Quality only affects lossy targets (JPEG/WebP); disable the control
  // for lossless ones so the form state matches what Resize actually does.
  const qualityDisabled = () => format() !== "jpg" && format() !== "webp";

  // Output size the current inputs resolve to for the reference image.
  function previewSize(): string {
    const src = firstSupported();
    if (!src) return "—";
    if (mode() === "percent") {
      const p = Math.min(Math.max(percent(), 1), 400);
      return `${Math.max(1, Math.round((src.width * p) / 100))}×${Math.max(1, Math.round((src.height * p) / 100))}`;
    }
    const w = width();
    const h = height();
    if (w <= 0 && h <= 0) return `${src.width}×${src.height}`;
    if (lock() && w <= 0 !== h <= 0) {
      if (w > 0) return `${w}×${Math.max(1, Math.round((src.height * w) / src.width))}`;
      return `${Math.max(1, Math.round((src.width * h) / src.height))}×${h}`;
    }
    if (w > 0 && h > 0) return `${w}×${h}`;
    return `${w > 0 ? w : src.width}×${h > 0 ? h : src.height}`;
  }

  function updateRow(path: string, patch: (row: Row) => Row) {
    setRows((prev) => prev.map((r) => (r.info.path === path ? patch(r) : r)));
  }

  async function pick() {
    if (running()) return;
    setNotice("");
    const paths = await ImageResize.pickImages();
    if (!paths || paths.length === 0) return;
    const probed = await probeAll(ImageResize.probe, paths);
    setRows(
      probed.map((info) => ({
        info,
        state: info.supported ? "pending" : "skipped",
        result: null,
        error: "",
      })),
    );
    const first = probed.find((i) => i.supported && i.width > 0);
    if (first) {
      // Default output format: keep each image's own format; the select
      // label pair below just tracks the reference image's format.
      setFormat(sourceFormat(first.path, first.format) as ImgFormat);
      setWidth(first.width);
      setHeight(first.height);
    }
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
        const res = await ImageResize.resize({
          path: r.info.path,
          targetFormat: format(),
          quality: quality(),
          mode: mode(),
          percent: percent(),
          width: width(),
          height: height(),
          lock: lock(),
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
    if (stopRequested) setNotice("已取消，剩余图片保持待处理");
    else if (anyDone) setNotice("全部完成，输出保存在原图所在文件夹");
  }

  async function save(row: Row) {
    if (!row.result) return;
    const dest = await ImageResize.saveAs(row.result.outputPath, fileName(row.result.outputPath));
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
              ? "按设定尺寸输出到原图旁；只转格式不改尺寸请用「图片格式转换」工具"
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
                <Label class="text-xs text-muted-foreground">输出格式</Label>
                <Select
                  options={formatOptions(supported())}
                  value={FORMAT_LABELS[format()]}
                  onChange={(label) => {
                    if (label && LABEL_BY_FORMAT[label])
                      setFormat(LABEL_BY_FORMAT[label] as ImgFormat);
                  }}
                  itemComponent={(props) => (
                    <SelectItem item={props.item}>{props.item.rawValue}</SelectItem>
                  )}
                >
                  <SelectTrigger class="w-44" aria-label="输出格式">
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
                  disabled={qualityDisabled()}
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

            {/* 尺寸 */}
            <div class="flex flex-col gap-3 border-t pt-3">
              <div class="flex flex-wrap items-center gap-x-4 gap-y-2">
                <Label class="text-xs text-muted-foreground">调整方式</Label>
                <div class="flex items-center gap-3 text-sm">
                  <label class="flex cursor-pointer items-center gap-1">
                    <input
                      type="radio"
                      name="resize-mode"
                      checked={mode() === "percent"}
                      onChange={() => setMode("percent")}
                    />
                    按比例
                  </label>
                  <label class="flex cursor-pointer items-center gap-1">
                    <input
                      type="radio"
                      name="resize-mode"
                      checked={mode() === "exact"}
                      onChange={() => setMode("exact")}
                    />
                    按尺寸
                  </label>
                </div>
                <span class="text-xs text-muted-foreground">
                  参考第 1 张{" "}
                  {firstSupported() ? `${firstSupported()!.width}×${firstSupported()!.height}` : ""}
                  → 输出 {previewSize()}
                </span>
              </div>

              <Show
                when={mode() === "percent"}
                fallback={
                  <div class="flex flex-wrap items-end gap-x-6 gap-y-3">
                    <div class="flex flex-col gap-1">
                      <Label class="text-xs text-muted-foreground">宽度（px）</Label>
                      <TextField
                        value={String(width())}
                        onChange={(v: string) => setWidth(Math.max(0, parseInt(v) || 0))}
                      >
                        <TextFieldInput class="w-28" type="number" min={0} step={10} />
                      </TextField>
                    </div>
                    <div class="flex flex-col gap-1">
                      <Label class="text-xs text-muted-foreground">高度（px）</Label>
                      <TextField
                        value={String(height())}
                        onChange={(v: string) => setHeight(Math.max(0, parseInt(v) || 0))}
                      >
                        <TextFieldInput class="w-28" type="number" min={0} step={10} />
                      </TextField>
                    </div>
                    <label class="flex cursor-pointer items-center gap-1 pb-1 text-sm">
                      <input
                        type="checkbox"
                        checked={lock()}
                        onChange={(e) => setLock(e.currentTarget.checked)}
                      />
                      锁定比例
                      <InfoTip label="锁定比例说明">
                        勾选时只填宽或高，另一边按原图比例自动算出（支持放大）；两边都填则完全按输入尺寸输出
                      </InfoTip>
                    </label>
                  </div>
                }
              >
                <div class="flex flex-wrap items-center gap-3">
                  <TextField
                    value={String(percent())}
                    onChange={(v: string) =>
                      setPercent(Math.min(400, Math.max(1, parseInt(v) || 1)))
                    }
                  >
                    <TextFieldInput class="w-24" type="number" min={1} max={400} />
                  </TextField>
                  <span class="text-sm text-muted-foreground">%（1–400，可放大）</span>
                </div>
              </Show>
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
                            {r.state === "skipped" ? r.info.skipReason : r.error}
                          </p>
                        </Show>
                      </div>
                      <div class="flex shrink-0 items-center gap-2">
                        <span class={`text-xs ${STATE_COLORS[r.state]}`}>
                          {STATE_LABELS[r.state]}
                        </span>
                        <Show when={r.state === "done" && r.result}>
                          <MotionButton variant="outline" size="sm" onClick={() => save(r)}>
                            保存到…
                          </MotionButton>
                          <MotionButton
                            variant="outline"
                            size="sm"
                            onClick={() => ImageResize.openFolder(r.result!.outputPath)}
                          >
                            打开文件夹
                          </MotionButton>
                        </Show>
                      </div>
                    </div>
                    <Show when={r.state === "done" && r.result?.previewDataURL}>
                      <img
                        class="max-h-40 self-start rounded-md"
                        src={r.result!.previewDataURL}
                        alt="调整预览"
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
                  <MotionButton
                    variant="outline"
                    onClick={() => {
                      stopRequested = true;
                      ImageResize.cancelActive();
                    }}
                  >
                    取消
                  </MotionButton>
                }
              >
                <MotionButton onClick={run} disabled={!hasPending()}>
                  开始调整
                </MotionButton>
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
