import type { JSX, ValidComponent } from "solid-js";
import { Show, splitProps } from "solid-js";
import type { PolymorphicProps } from "@kobalte/core/polymorphic";

import { Button, type ButtonProps } from "~/components/ui/button";
import { cn } from "~/lib/utils";

/** Shared enter-animation utilities, gated on motion safety so users with
 * "reduce motion" set at the OS level get instant swaps. tailwindcss-animate
 * supplies animate-in / fade-in / zoom-in / slide-in-from-*. */
export const ENTER = "motion-safe:animate-in motion-safe:fade-in motion-safe:duration-200";

/** Enter animation for full view swaps: fade plus a slight upward slide. */
export const ENTER_VIEW = `${ENTER} motion-safe:zoom-in-95 motion-safe:slide-in-from-bottom-2`;

/**
 * MotionButton is Button with press feedback: it scales down slightly
 * while pressed and eases back on release. All Button props pass through.
 */
export function MotionButton<T extends ValidComponent = "button">(
  props: PolymorphicProps<T, ButtonProps<T>>,
) {
  const [local, others] = splitProps(props as ButtonProps, ["class"]);
  return (
    <Button
      class={cn(
        "motion-safe:transition-transform motion-safe:duration-150 motion-safe:ease-out motion-safe:active:scale-[0.97]",
        local.class,
      )}
      {...others}
    />
  );
}

/**
 * FadeSwap replays an enter animation whenever `when` flips: Solid's keyed
 * <Show> remounts the subtree, so the CSS keyframe starts on insertion.
 * Use for one-shot view replacements (tool grid vs an open tool page).
 */
export function FadeSwap(props: { when: unknown; class?: string; children: JSX.Element }) {
  return (
    <Show when={props.when} keyed>
      <div class={cn(ENTER_VIEW, props.class)}>{props.children}</div>
    </Show>
  );
}
