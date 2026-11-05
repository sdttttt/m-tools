import { createMemo, createSignal, For, Show } from "solid-js";

/** Shortest selectable clip, in seconds. */
export const MIN_TRIM_SPAN = 0.1;

/** Formats seconds as mm:ss.s, video-editor style. */
export function formatTimecode(sec: number): string {
  const s = Math.max(0, sec);
  const minutes = Math.floor(s / 60);
  const rest = s - minutes * 60;
  return `${String(minutes).padStart(2, "0")}:${rest.toFixed(1).padStart(4, "0")}`;
}

function clamp(value: number, lo: number, hi: number): number {
  return Math.min(hi, Math.max(lo, value));
}

function roundTenth(value: number): number {
  return Math.round(value * 10) / 10;
}

type DragMode = "start" | "end" | "move";

export type TrimTimelineProps = {
  /** Filmstrip frames as image URLs; may be empty while loading. */
  frames: string[];
  /** Full clip length in seconds. */
  duration: number;
  start: number;
  end: number;
  loading?: boolean;
  disabled?: boolean;
  onChange: (start: number, end: number) => void;
};

/**
 * Trim bar shaped like a video editor's timeline: a filmstrip of the clip
 * with draggable handles that pick the preserved range. Handles and the
 * selection respond to the mouse, touch and arrow keys (shift = 1s).
 */
export function TrimTimeline(props: TrimTimelineProps) {
  const [track, setTrack] = createSignal<HTMLDivElement>();

  const percent = (sec: number) =>
    props.duration > 0 ? clamp(sec / props.duration, 0, 1) * 100 : 0;
  const startPct = createMemo(() => percent(props.start));
  const endPct = createMemo(() => percent(props.end));
  const span = () => Math.max(0, props.end - props.start);

  function beginDrag(mode: DragMode, event: PointerEvent) {
    const el = track();
    if (props.disabled || !el || props.duration <= 0) return;
    event.preventDefault();
    if (mode !== "move") (event.currentTarget as HTMLElement).focus();
    const rect = el.getBoundingClientRect();
    if (rect.width <= 0) return;

    const secPerPx = props.duration / rect.width;
    const x0 = event.clientX;
    const start0 = props.start;
    const end0 = props.end;

    const onMove = (ev: PointerEvent) => {
      const delta = (ev.clientX - x0) * secPerPx;
      if (mode === "start") {
        props.onChange(clamp(roundTenth(start0 + delta), 0, end0 - MIN_TRIM_SPAN), end0);
      } else if (mode === "end") {
        props.onChange(
          start0,
          clamp(roundTenth(end0 + delta), start0 + MIN_TRIM_SPAN, props.duration),
        );
      } else {
        const width = end0 - start0;
        const next = clamp(roundTenth(start0 + delta), 0, props.duration - width);
        props.onChange(next, roundTenth(next + width));
      }
    };
    const onUp = () => {
      window.removeEventListener("pointermove", onMove);
      window.removeEventListener("pointerup", onUp);
      window.removeEventListener("pointercancel", onUp);
    };
    window.addEventListener("pointermove", onMove);
    window.addEventListener("pointerup", onUp);
    window.addEventListener("pointercancel", onUp);
  }

  function nudge(edge: "start" | "end", delta: number) {
    if (props.disabled || props.duration <= 0) return;
    if (edge === "start") {
      props.onChange(clamp(props.start + delta, 0, props.end - MIN_TRIM_SPAN), props.end);
    } else {
      props.onChange(
        props.start,
        clamp(props.end + delta, props.start + MIN_TRIM_SPAN, props.duration),
      );
    }
  }

  const onHandleKeyDown = (edge: "start" | "end") => (event: KeyboardEvent) => {
    const step = event.shiftKey ? 1 : 0.1;
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      nudge(edge, -step);
    } else if (event.key === "ArrowRight") {
      event.preventDefault();
      nudge(edge, step);
    }
  };

  return (
    <div class="flex flex-col gap-2">
      <div class="flex items-baseline justify-between gap-3 text-xs">
        <span class="text-muted-foreground">拖动两端手柄截取片段</span>
        <span class="tabular-nums text-muted-foreground">
          {formatTimecode(props.start)} – {formatTimecode(props.end)} · 共 {formatTimecode(span())}
        </span>
      </div>
      <div
        ref={setTrack}
        class="relative h-14 touch-none select-none overflow-hidden rounded-md border bg-muted"
        classList={{ "opacity-60": !!props.disabled }}
      >
        <div class="pointer-events-none absolute inset-0 flex">
          <Show when={props.frames.length > 0} fallback={<div class="flex-1" />}>
            <For each={props.frames}>
              {(src) => (
                <img
                  src={src}
                  alt=""
                  draggable={false}
                  class="h-full min-w-0 flex-1 object-cover"
                />
              )}
            </For>
          </Show>
        </div>

        <Show when={props.loading && props.frames.length === 0}>
          <span class="absolute inset-0 grid place-items-center text-xs text-muted-foreground">
            生成缩略图…
          </span>
        </Show>

        <div
          class="pointer-events-none absolute inset-y-0 left-0 bg-background/70"
          style={{ width: `${startPct()}%` }}
        />
        <div
          class="pointer-events-none absolute inset-y-0 right-0 bg-background/70"
          style={{ width: `${100 - endPct()}%` }}
        />

        <div
          class="absolute inset-y-0 cursor-grab active:cursor-grabbing"
          style={{ left: `${startPct()}%`, width: `${endPct() - startPct()}%` }}
          onPointerDown={(event) => beginDrag("move", event)}
        >
          <div class="pointer-events-none absolute inset-0 border-y-2 border-primary" />
          <div class="pointer-events-none absolute inset-y-0 left-0 w-px bg-primary" />
          <div class="pointer-events-none absolute inset-y-0 right-0 w-px bg-primary" />
        </div>

        <button
          type="button"
          role="slider"
          aria-label="截取起点"
          aria-valuemin={0}
          aria-valuemax={props.duration}
          aria-valuenow={props.start}
          aria-valuetext={formatTimecode(props.start)}
          class="absolute inset-y-0 z-10 w-1.5 cursor-ew-resize border-0 bg-primary p-0 outline-none focus-visible:ring-2 focus-visible:ring-ring"
          style={{ left: `${startPct()}%` }}
          onPointerDown={(event) => beginDrag("start", event)}
          onKeyDown={onHandleKeyDown("start")}
        />
        <button
          type="button"
          role="slider"
          aria-label="截取终点"
          aria-valuemin={0}
          aria-valuemax={props.duration}
          aria-valuenow={props.end}
          aria-valuetext={formatTimecode(props.end)}
          class="absolute inset-y-0 z-10 w-1.5 cursor-ew-resize border-0 bg-primary p-0 outline-none focus-visible:ring-2 focus-visible:ring-ring"
          style={{ right: `${100 - endPct()}%` }}
          onPointerDown={(event) => beginDrag("end", event)}
          onKeyDown={onHandleKeyDown("end")}
        />
      </div>
    </div>
  );
}
