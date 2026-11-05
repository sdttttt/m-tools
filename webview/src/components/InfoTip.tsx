import type { JSX } from "solid-js";

import { Tooltip, TooltipContent, TooltipTrigger } from "~/components/ui/tooltip";

interface InfoTipProps {
  class?: string | undefined;
  label: string;
  children: JSX.Element;
}

/** Circle-help icon that reveals `children` in a tooltip on hover/focus. */
export function InfoTip(props: InfoTipProps) {
  return (
    <Tooltip placement="top">
      <TooltipTrigger
        as="span"
        class={props.class ?? "cursor-help select-none text-muted-foreground"}
        aria-label={props.label}
        tabindex="0"
      >
        <svg
          xmlns="http://www.w3.org/2000/svg"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="2"
          stroke-linecap="round"
          stroke-linejoin="round"
          class="size-3.5"
        >
          <circle cx="12" cy="12" r="10" />
          <path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3" />
          <path d="M12 17h.01" />
        </svg>
      </TooltipTrigger>
      <TooltipContent class="max-w-56 text-xs leading-4">{props.children}</TooltipContent>
    </Tooltip>
  );
}
