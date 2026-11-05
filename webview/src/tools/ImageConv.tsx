import { createSignal, For, Show } from "solid-js";
import { ImageConv, type ImageInfo, type ImgconvConvertResult } from "~/mygo";
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
import { TextField, TextFieldInput } from "~/components/ui/text-field";

type Quality = "high" | "balanced" | "compact";

type RowState = "pending" | "converting" | "done" | "failed" | "skipped";

interface Row {
  info: ImageInfo;
  state: RowState;
  result: ImgconvConvertResult | null;
  error: string;
}

const TARGET_LABELS: Record<string, string> = {
  jpg: "JPEG（.jpg）",
  png: "PNG（.png）",
  gif: "GIF（.gif）",
  webp: "WebP（.webp）",
  bmp: "BMP（.bmp）",
  tiff: "TIFF（.tiff）",
};

const TARGET_BY_LABEL: Record<string, string> = Object.fromEntries(
  Object.entries(TARGET_LABELS).map(([id, label]) => [label, id]),
);

// targetOptions lists display labels; WebP drops out entirely on builds
// without the codec (options/value/onChange all live in label-land).
function targetOptions(supported: boolean): string[] {
  return Object.entries(TARGET_LABELS)
    .filter(([id]) => id !== "webp" || supported)
    .map(([, label]) => label);
}

const QUALITY_LABELS: Record<Quality, string> = {
  high: "高画质",
  balanced: "均衡",
  compact: "紧凑（体积小）",
};

const QUALITY_BY_LABEL: Record<string, Quality> = {
  高画质: "high",
  均衡: "balanced",
  "紧凑（体积小）": "compact",
};

const STATE_LABELS: Record<RowState, string> = {
  pending: "待转换",
  converting: "转换中…",
  done: "完成",
  failed: "失败",
  skipped: "跳过",
};

// The page treats tif/tiff and jpg/jpeg as one target: the Go side always
// writes .tiff / .jpg.
function sourceFormat(path: string, probe?: ImageInfo): string {
  if (probe?.format) return probe.format;
  const ext = path.split(".").pop()?.toLowerCase() ?? "";
  if (ext === "jpeg") return "jpg";
  if (ext === "tif") return "tiff";
  return ext;
}

function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1048576) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1048576).toFixed(1)} MB`;
}

export default function ImageConvTool() {
  const [supported, setSupported] = createSignal(true);
  const [rows, setRows] = createSignal<Row[]>([]);
  const [target, setTarget] = createSignal("png");
  const [quality, setQuality] = createSignal<Quality>("balanced");
  const [maxWidth, setMaxWidth] = createSignal(0);
  const [running, setRunning] = createSignal(false);
  const [notice, setNotice] = createSignal("");
  let stopRequested = false;

  ImageConv.supported()
    .then(setSupported)
    .catch(() => setSupported(false));

  function updateRow(path: string, patch: (row: Row) => Row) {
    setRows((prev) => prev.map((r) => (r.info.path === path ? patch(r) : r)));
  }

  async function pick() {
    if (running()) return;
    setNotice("");
    const paths = await ImageConv.pickImages();
    if (!paths || paths.length === 0) return;
    const probed = await Promise.all(
      paths.map(async (p) => {
        try {
          return await ImageConv.probe(p);
        } catch {
          return { path: p, format: "", supported: false, skipReason: "读取失败" } as ImageInfo;
        }
      }),
    );
    setRows(
      probed.map((info) => {
        const fmt = sourceFormat(info.path, info);
        const same = fmt === target() && info.supported;
        return {
          info,
          state: same ? "skipped" : info.supported ? "pending" : "skipped",
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
        const fmt = sourceFormat(r.info.path, r.info);
        const same = fmt === target() && r.info.supported;
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
          maxWidth: maxWidth(),
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
    const name = row.result.outputPath.split("/").pop() ?? row.result.outputPath;
    const dest = await ImageConv.saveAs(row.result.outputPath, name);
    if (dest) setNotice(`已另存到 ${dest}`);
  }

  const hasPending = () => rows().some((r) => r.state === "pending");
  const previewRow = () => rows().find((r) => r.state === "done" && r.result?.previewDataURL);

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
              ? "转换在原图所在文件夹生成新文件，不动原图"
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
                  options={targetOptions(supported())}
                  value={TARGET_LABELS[target()]}
                  onChange={(label) => {
                    if (!label) return;
                    setTarget(TARGET_BY_LABEL[label] ?? target());
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
                <Label class="text-xs text-muted-foreground">画质（仅影响 JPEG / WebP）</Label>
                <Select
                  options={Object.values(QUALITY_LABELS)}
                  value={QUALITY_LABELS[quality()]}
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
              <div class="flex flex-col gap-1">
                <div class="flex items-center gap-1">
                  <Label class="text-xs text-muted-foreground">最大宽度（0 = 原尺寸）</Label>
                  <InfoTip label="最大宽度说明">
                    等比缩放、只缩小不放大。例：4000×3000 的图填 800 → 输出 800×600；填 0 保持原尺寸
                  </InfoTip>
                </div>
                <TextField
                  value={String(maxWidth())}
                  onChange={(value: string) => setMaxWidth(Math.max(0, parseInt(value) || 0))}
                >
                  <TextFieldInput class="w-32" type="number" min={0} step={100} />
                </TextField>
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
                          {r.info.path.split("/").pop()}
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
                                `与目标格式相同（${sourceFormat(r.info.path, r.info).toUpperCase()}）`
                              : r.error}
                          </p>
                        </Show>
                      </div>
                      <div class="flex shrink-0 items-center gap-2">
                        <span
                          class="text-xs"
                          classList={{
                            "text-muted-foreground":
                              r.state === "pending" || r.state === "converting",
                            "text-primary": r.state === "done",
                            "text-destructive": r.state === "failed" || r.state === "skipped",
                          }}
                        >
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
                <Button onClick={run} disabled={!hasPending() || rows().length === 0}>
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

      {/* 最近一次完成的转换预览（列表为空时，例如重新选择文件后） */}
      <Show when={rows().length === 0 && previewRow()}>
        {(r) => (
          <Card>
            <CardContent class="flex flex-col gap-2 px-5 py-4">
              <img
                class="mx-auto max-h-64 rounded-md"
                src={r().result!.previewDataURL}
                alt="预览"
              />
              <p class="m-0 text-center text-xs text-muted-foreground">
                {r().result!.format.toUpperCase()} · {fmtSize(r().result!.sizeBytes)}
              </p>
            </CardContent>
          </Card>
        )}
      </Show>
    </div>
  );
}
