import { createSignal, onCleanup, Show } from "solid-js";
import { Mp4Gif, events, type ConvertResult, type VideoInfo } from "../mygo";

type Phase = "idle" | "converting" | "done" | "error";

const QUALITIES = [
  { value: "high", label: "高画质" },
  { value: "balanced", label: "均衡" },
  { value: "compact", label: "紧凑（体积小）" },
] as const;

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
  const [quality, setQuality] = createSignal("balanced");
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

  const field =
    "w-24 rounded border border-line bg-card px-2 py-1 text-sm text-fg outline-none focus:border-accent";

  return (
    <div class="mx-auto flex w-full max-w-xl flex-col gap-5">
      {/* 文件选择 */}
      <button
        class="cursor-pointer rounded-xl border border-dashed border-line bg-card px-6 py-5 text-left hover:border-accent"
        onClick={pick}
        disabled={converting()}
      >
        <p class="m-0 text-sm">{videoPath() || "点击选择视频文件（MP4 / MOV / MKV / WebM…）"}</p>
        <Show when={info()}>
          {(v) => (
            <p class="m-0 mt-1 text-xs text-muted">
              {v().width}×{v().height} · {v().fps.toFixed(1)} fps · {v().videoCodec} ·{" "}
              {fmtSec(v().durationSec)} · {fmtSize(v().sizeBytes)}
            </p>
          )}
        </Show>
      </button>

      {/* 参数 */}
      <Show when={info()}>
        <div class="flex flex-wrap items-end gap-x-6 gap-y-3 rounded-xl border border-line bg-card px-5 py-4 text-sm">
          <label class="flex flex-col gap-1">
            <span class="text-xs text-muted">起始秒</span>
            <input
              class={field}
              placeholder="从头"
              value={startSec()}
              onInput={(e) => setStartSec(e.currentTarget.value)}
            />
          </label>
          <label class="flex flex-col gap-1">
            <span class="text-xs text-muted">时长秒</span>
            <input
              class={field}
              placeholder="到结尾"
              value={durationSec()}
              onInput={(e) => setDurationSec(e.currentTarget.value)}
            />
          </label>
          <label class="flex flex-col gap-1">
            <span class="text-xs text-muted">帧率</span>
            <input
              class={field}
              type="number"
              min={5}
              max={30}
              value={fps()}
              onInput={(e) => setFps(parseInt(e.currentTarget.value) || 15)}
            />
          </label>
          <label class="flex flex-col gap-1">
            <span class="text-xs text-muted">最大宽度</span>
            <input
              class={field}
              type="number"
              min={100}
              max={1920}
              value={maxWidth()}
              onInput={(e) => setMaxWidth(parseInt(e.currentTarget.value) || 480)}
            />
          </label>
          <label class="flex flex-col gap-1">
            <span class="text-xs text-muted">画质</span>
            <select class={field} onChange={(e) => setQuality(e.currentTarget.value)}>
              {QUALITIES.map((q) => (
                <option value={q.value} selected={quality() === q.value}>
                  {q.label}
                </option>
              ))}
            </select>
          </label>
        </div>
      </Show>

      {/* 操作 */}
      <div class="flex items-center gap-3">
        <Show
          when={converting()}
          fallback={
            <button
              class="cursor-pointer rounded-lg bg-accent px-4 py-2 text-sm text-white disabled:cursor-not-allowed disabled:opacity-40"
              onClick={convert}
              disabled={!info()}
            >
              开始转换
            </button>
          }
        >
          <button
            class="cursor-pointer rounded-lg border border-line bg-card px-4 py-2 text-sm"
            onClick={() => Mp4Gif.cancelActive()}
          >
            取消
          </button>
          <span class="text-sm text-muted">
            {STAGE_LABELS[progress().stage] ?? "准备中"} · {percent()}%
          </span>
        </Show>
      </div>

      {/* 进度条 */}
      <Show when={converting()}>
        <div class="h-1.5 w-full overflow-hidden rounded-full bg-card">
          <div class="h-full bg-accent transition-[width]" style={{ width: `${percent()}%` }} />
        </div>
      </Show>

      {/* 结果 */}
      <Show when={result()}>
        {(r) => (
          <div class="flex flex-col gap-3 rounded-xl border border-line bg-card px-5 py-4">
            <Show when={r().previewDataURL}>
              <img class="mx-auto max-h-64 rounded" src={r().previewDataURL} alt="GIF 预览" />
            </Show>
            <p class="m-0 text-sm">
              {r().width}×{r().height} · {r().frames} 帧 · {fmtSize(r().sizeBytes)} · 耗时{" "}
              {(r().durationMs / 1000).toFixed(1)} 秒
            </p>
            <div class="flex items-center gap-3">
              <button
                class="cursor-pointer rounded-lg bg-accent px-4 py-2 text-sm text-white"
                onClick={save}
              >
                保存到…
              </button>
              <Show when={savedPath()}>
                <button
                  class="cursor-pointer rounded-lg border border-line bg-card px-4 py-2 text-sm"
                  onClick={() => Mp4Gif.openFolder(savedPath())}
                >
                  打开文件夹
                </button>
                <span class="truncate text-xs text-muted">{savedPath()}</span>
              </Show>
            </div>
          </div>
        )}
      </Show>

      {/* 错误 */}
      <Show when={error()}>
        <p class="m-0 rounded-lg border border-line bg-card px-4 py-3 text-sm text-accent">
          {error()}
        </p>
      </Show>
    </div>
  );
}
