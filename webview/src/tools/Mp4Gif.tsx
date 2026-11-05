import { createEffect, createSignal, on, onCleanup, Show } from "solid-js";
import { Mp4Gif, events, type ConvertResult, type PreviewResult, type VideoInfo } from "~/mygo";
import { Button } from "~/components/ui/button";
import { Card, CardContent } from "~/components/ui/card";
import { MIN_TRIM_SPAN, TrimTimeline } from "~/components/TrimTimeline";
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
  const [trimStart, setTrimStart] = createSignal(0);
  const [trimEnd, setTrimEnd] = createSignal(0);
  const [thumbs, setThumbs] = createSignal<string[]>([]);
  const [thumbLoading, setThumbLoading] = createSignal(false);
  const [fps, setFps] = createSignal(8);
  const [maxWidth, setMaxWidth] = createSignal(240);
  const [quality, setQuality] = createSignal<Quality>("compact");
  const [phase, setPhase] = createSignal<Phase>("idle");
  const [progress, setProgress] = createSignal({ stage: "", percent: 0 });
  const [result, setResult] = createSignal<ConvertResult | null>(null);
  const [error, setError] = createSignal("");
  const [savedPath, setSavedPath] = createSignal("");
  const [preview, setPreview] = createSignal<PreviewResult | null>(null);
  const [previewing, setPreviewing] = createSignal(false);
  let previewSeq = 0; // bumped whenever the preview's inputs change

  const unsubscribe = events.mp4gifProgress.on((p) =>
    setProgress({ stage: p.stage, percent: p.percent }),
  );
  onCleanup(unsubscribe);

  const suggestedName = () => videoPath().replace(/\.[^.]+$/, "") + ".gif";
  const converting = () => phase() === "converting";
  const percent = () => Math.round(progress().percent);
  const clipDuration = () => info()?.durationSec ?? 0;
  const secondsInput = (v: number) => String(Number(v.toFixed(1)));

  function setStart(value: number) {
    setTrimStart(Math.min(Math.max(0, value), Math.max(0, trimEnd() - MIN_TRIM_SPAN)));
  }

  function setEnd(value: number) {
    const min = trimStart() + MIN_TRIM_SPAN;
    const max = clipDuration() > 0 ? clipDuration() : Math.max(min, value);
    setTrimEnd(Math.min(Math.max(min, value), max));
  }

  // A preview only describes the range and settings it was rendered with.
  createEffect(
    on(
      [trimStart, trimEnd, fps, maxWidth, quality],
      () => {
        previewSeq++;
        setPreview(null);
      },
      { defer: true },
    ),
  );

  async function pick() {
    const p = await Mp4Gif.pickVideo();
    if (!p) return;
    setPhase("idle");
    setResult(null);
    setError("");
    setSavedPath("");
    setInfo(null);
    setThumbs([]);
    setTrimStart(0);
    setTrimEnd(0);
    setVideoPath(p);
    previewSeq++;
    setPreview(null);
    try {
      const v = await Mp4Gif.probe(p);
      setInfo(v);
      setTrimEnd(v.durationSec);
      setFps(Math.max(8, Math.min(12, Math.floor(v.fps) || 8)));
      setMaxWidth(Math.min(240, Math.max(100, v.width)));
      setThumbLoading(true);
      Mp4Gif.thumbnails(p)
        .then((frames) => {
          if (videoPath() === p) setThumbs(frames);
        })
        .catch(() => {})
        .finally(() => {
          if (videoPath() === p) setThumbLoading(false);
        });
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
        startSec: trimStart(),
        durationSec: Math.max(0, trimEnd() - trimStart()),
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

  async function previewClip() {
    if (!info()) return;
    const seq = ++previewSeq;
    setPreviewing(true);
    setError("");
    try {
      const res = await Mp4Gif.preview({
        path: videoPath(),
        startSec: trimStart(),
        durationSec: Math.max(0, trimEnd() - trimStart()),
        fps: fps(),
        maxWidth: maxWidth(),
        quality: quality(),
      });
      if (seq === previewSeq) setPreview(res);
    } catch (e) {
      const message = String(e).replace(/^Error:\s*/, "");
      if (message !== "预览已取消") setError(message);
    } finally {
      setPreviewing(false);
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
          <CardContent class="flex flex-col gap-4 px-5 py-4">
            <TrimTimeline
              frames={thumbs()}
              duration={clipDuration()}
              start={trimStart()}
              end={trimEnd()}
              loading={thumbLoading()}
              disabled={converting()}
              onChange={(start, end) => {
                setTrimStart(start);
                setTrimEnd(end);
              }}
            />
            <div class="flex flex-wrap items-end gap-x-6 gap-y-3">
              <div class="flex flex-col gap-1">
                <Label class="text-xs text-muted-foreground">起点秒</Label>
                <TextField
                  value={secondsInput(trimStart())}
                  onChange={(value: string) => setStart(parseFloat(value) || 0)}
                >
                  <TextFieldInput class="w-24" type="number" min={0} step={0.1} />
                </TextField>
              </div>
              <div class="flex flex-col gap-1">
                <Label class="text-xs text-muted-foreground">终点秒</Label>
                <TextField
                  value={secondsInput(trimEnd())}
                  onChange={(value: string) => setEnd(parseFloat(value) || 0)}
                >
                  <TextFieldInput class="w-24" type="number" min={0} step={0.1} />
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
            </div>

            {/* 片段预览 */}
            <Show when={preview()}>
              {(p) => (
                <div class="flex flex-col gap-2 border-t pt-3">
                  <img class="mx-auto max-h-56 rounded-md" src={p().dataURL} alt="片段预览" />
                  <p class="m-0 text-center text-xs text-muted-foreground">
                    预览 {p().durationSec.toFixed(1)} 秒 · {p().width}×{p().height} · {p().frames}{" "}
                    帧 · {fmtSize(p().sizeBytes)}
                  </p>
                </div>
              )}
            </Show>
          </CardContent>
        </Card>
      </Show>

      {/* 操作 */}
      <div class="flex items-center gap-3">
        <Show
          when={converting()}
          fallback={
            <>
              <Button onClick={convert} disabled={!info() || previewing()}>
                开始转换
              </Button>
              <Show
                when={previewing()}
                fallback={
                  <Button variant="outline" onClick={previewClip} disabled={!info()}>
                    预览片段
                  </Button>
                }
              >
                <Button variant="outline" onClick={() => Mp4Gif.cancelActive()}>
                  取消预览
                </Button>
                <span class="text-sm text-muted-foreground">生成预览…</span>
              </Show>
            </>
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
