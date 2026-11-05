import { createSignal, onCleanup, Show } from "solid-js";
import { Mp4Gif, events, type ConvertResult, type VideoInfo } from "~/mygo";
import { Button } from "~/components/ui/button";
import { Card, CardContent } from "~/components/ui/card";
import { Label } from "~/components/ui/label";
import { Progress } from "~/components/ui/progress";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "~/components/ui/select";
import { TextField, TextFieldInput } from "~/components/ui/text-field";

type Phase = "idle" | "converting" | "done" | "error";

type Quality = "high" | "balanced" | "compact";

const QUALITY_LABELS = ["高画质", "均衡", "紧凑（体积小）"] as const;

const QUALITY_BY_LABEL: Record<string, Quality> = {
  高画质: "high",
  均衡: "balanced",
  "紧凑（体积小）": "compact",
};

const LABEL_BY_QUALITY: Record<Quality, string> = {
  high: "高画质",
  balanced: "均衡",
  compact: "紧凑（体积小）",
};

const STAGE_LABELS: Record<string, string> = {
  palette: "分析调色板",
  encode: "编码 GIF",
};

function fmtSize(n: number): string {
  if (n < 1024) return `${n} B`;
  if (n < 1048576) return `${(n / 1024).toFixed(1)} KB`;
  return `${(n / 1048576).toFixed(1)} MB`;
}

function fmtSec(n: number): string {
  return n >= 60 ? `${Math.floor(n / 60)} 分 ${(n % 60).toFixed(0)} 秒` : `${n.toFixed(1)} 秒`;
}

export default function Mp4GifTool() {
  const [videoPath, setVideoPath] = createSignal("");
  const [info, setInfo] = createSignal<VideoInfo | null>(null);
  const [startSec, setStartSec] = createSignal("");
  const [durationSec, setDurationSec] = createSignal("");
  const [fps, setFps] = createSignal(15);
  const [maxWidth, setMaxWidth] = createSignal(480);
  const [quality, setQuality] = createSignal<Quality>("balanced");
  const [phase, setPhase] = createSignal<Phase>("idle");
  const [progress, setProgress] = createSignal({ stage: "", percent: 0 });
  const [result, setResult] = createSignal<ConvertResult | null>(null);
  const [error, setError] = createSignal("");
  const [savedPath, setSavedPath] = createSignal("");

  const unsubscribe = events.mp4gifProgress.on((p) =>
    setProgress({ stage: p.stage, percent: p.percent }),
  );
  onCleanup(unsubscribe);

  const suggestedName = () => videoPath().replace(/\.[^.]+$/, "") + ".gif";
  const converting = () => phase() === "converting";
  const percent = () => Math.round(progress().percent);

  async function pick() {
    const p = await Mp4Gif.pickVideo();
    if (!p) return;
    setPhase("idle");
    setResult(null);
    setError("");
    setSavedPath("");
    setInfo(null);
    setVideoPath(p);
    try {
      const v = await Mp4Gif.probe(p);
      setInfo(v);
      setFps(Math.max(5, Math.min(15, Math.floor(v.fps) || 15)));
      setMaxWidth(Math.min(480, Math.max(100, v.width)));
    } catch (e) {
      setError(String(e));
      setPhase("error");
    }
  }

  async function convert() {
    setPhase("converting");
    setError("");
    setProgress({ stage: "", percent: 0 });
    try {
      const res = await Mp4Gif.convert({
        path: videoPath(),
        startSec: parseFloat(startSec()) || 0,
        durationSec: parseFloat(durationSec()) || 0,
        fps: fps(),
        maxWidth: maxWidth(),
        quality: quality(),
      });
      setResult(res);
      setPhase("done");
    } catch (e) {
      setError(String(e).replace(/^Error:\s*/, ""));
      setPhase("error");
    }
  }

  async function save() {
    if (!result()) return;
    const dest = await Mp4Gif.saveAs(result()!.outputPath, suggestedName());
    if (dest) setSavedPath(dest);
  }

  return (
    <div class="mx-auto flex w-full max-w-xl flex-col gap-5">
      {/* 文件选择 */}
      <Card
        class="cursor-pointer border-dashed transition-colors hover:border-primary"
        onClick={() => !converting() && pick()}
      >
        <CardContent class="flex flex-col gap-1 px-6 py-5">
          <p class="m-0 text-sm">{videoPath() || "点击选择视频文件（MP4 / MOV / MKV / WebM…）"}</p>
          <Show when={info()}>
            {(v) => (
              <p class="m-0 text-xs text-muted-foreground">
                {v().width}×{v().height} · {v().fps.toFixed(1)} fps · {v().videoCodec} ·{" "}
                {fmtSec(v().durationSec)} · {fmtSize(v().sizeBytes)}
              </p>
            )}
          </Show>
        </CardContent>
      </Card>

      {/* 参数 */}
      <Show when={info()}>
        <Card>
          <CardContent class="flex flex-wrap items-end gap-x-6 gap-y-3 px-5 py-4">
            <div class="flex flex-col gap-1">
              <Label class="text-xs text-muted-foreground">起始秒</Label>
              <TextField value={startSec()} onChange={(value: string) => setStartSec(value)}>
                <TextFieldInput class="w-24" placeholder="从头" />
              </TextField>
            </div>
            <div class="flex flex-col gap-1">
              <Label class="text-xs text-muted-foreground">时长秒</Label>
              <TextField value={durationSec()} onChange={(value: string) => setDurationSec(value)}>
                <TextFieldInput class="w-24" placeholder="到结尾" />
              </TextField>
            </div>
            <div class="flex flex-col gap-1">
              <Label class="text-xs text-muted-foreground">帧率</Label>
              <TextField
                value={String(fps())}
                onChange={(value: string) => setFps(parseInt(value) || 15)}
              >
                <TextFieldInput class="w-24" type="number" min={5} max={30} />
              </TextField>
            </div>
            <div class="flex flex-col gap-1">
              <Label class="text-xs text-muted-foreground">最大宽度</Label>
              <TextField
                value={String(maxWidth())}
                onChange={(value: string) => setMaxWidth(parseInt(value) || 480)}
              >
                <TextFieldInput class="w-24" type="number" min={100} max={1920} />
              </TextField>
            </div>
            <div class="flex flex-col gap-1">
              <Label class="text-xs text-muted-foreground">画质</Label>
              <Select
                options={[...QUALITY_LABELS]}
                value={LABEL_BY_QUALITY[quality()]}
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
          </CardContent>
        </Card>
      </Show>

      {/* 操作 */}
      <div class="flex items-center gap-3">
        <Show
          when={converting()}
          fallback={
            <Button onClick={convert} disabled={!info()}>
              开始转换
            </Button>
          }
        >
          <Button variant="outline" onClick={() => Mp4Gif.cancelActive()}>
            取消
          </Button>
          <span class="text-sm text-muted-foreground">
            {STAGE_LABELS[progress().stage] ?? "准备中"} · {percent()}%
          </span>
        </Show>
      </div>

      {/* 进度条 */}
      <Show when={converting()}>
        <Progress value={percent()} class="w-full" />
      </Show>

      {/* 结果 */}
      <Show when={result()}>
        {(r) => (
          <Card>
            <CardContent class="flex flex-col gap-3 px-5 py-4">
              <Show when={r().previewDataURL}>
                <img class="mx-auto max-h-64 rounded-md" src={r().previewDataURL} alt="GIF 预览" />
              </Show>
              <p class="m-0 text-sm">
                {r().width}×{r().height} · {r().frames} 帧 · {fmtSize(r().sizeBytes)} · 耗时{" "}
                {(r().durationMs / 1000).toFixed(1)} 秒
              </p>
              <div class="flex items-center gap-3">
                <Button onClick={save}>保存到…</Button>
                <Show when={savedPath()}>
                  <Button variant="outline" onClick={() => Mp4Gif.openFolder(savedPath())}>
                    打开文件夹
                  </Button>
                  <span class="truncate text-xs text-muted-foreground">{savedPath()}</span>
                </Show>
              </div>
            </CardContent>
          </Card>
        )}
      </Show>

      {/* 错误 */}
      <Show when={error()}>
        <p class="m-0 rounded-md border border-destructive/50 bg-destructive/5 px-4 py-3 text-sm text-destructive">
          {error()}
        </p>
      </Show>
    </div>
  );
}
