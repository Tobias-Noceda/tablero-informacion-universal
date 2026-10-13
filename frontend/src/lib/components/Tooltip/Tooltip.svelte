<script lang="ts" module>
  type TooltipPlacement =
    | 'top'
    | 'bottom'
    | 'left'
    | 'right'
    | 'top-left'
    | 'bottom-left'
    | 'top-right'
    | 'bottom-right';

  type TooltipReference = 'mouse' | 'trigger';
</script>

<script lang="ts">
  import { cn, uuid } from '$lib/utils';
  import { portal } from '$lib/actions/portal';
  import type { Snippet } from 'svelte';

  interface Props {
    content: string | Snippet;
    children: Snippet;
    placement?: TooltipPlacement;
    reference?: TooltipReference;
    delay?: number;
    disabled?: boolean;
    class?: string;
    tooltipClass?: string;
  }

  let {
    content,
    children,
    placement = 'top',
    reference = 'trigger',
    delay = 100,
    disabled = false,
    class: triggerClass,
    tooltipClass
  }: Props = $props();

  const OFFSET = 8; // Distance from the reference (trigger or cursor)

  const id = `tooltip-${uuid()}`;

  let trigger: HTMLDivElement;
  let timeout: ReturnType<typeof setTimeout> | undefined;
  let mouse = { top: 0, left: 0 };

  let visible = $state(false);
  let position = $state({ top: 0, left: 0 });

  type Box = Pick<DOMRect, 'top' | 'bottom' | 'left' | 'right' | 'width' | 'height'>;

  // A zero-size box at the cursor, so every placement works the same around it
  function pointBox({ top, left }: typeof mouse): Box {
    return { top, bottom: top, left, right: left, width: 0, height: 0 };
  }

  function referenceBox(): Box {
    return reference === 'mouse' ? pointBox(mouse) : trigger.getBoundingClientRect();
  }

  function calculatePosition(rect: Box) {
    switch (placement) {
      case 'top':
        return { top: rect.top - OFFSET, left: rect.left + rect.width / 2 };
      // Corners sit diagonally outside the reference, anchored to that corner
      case 'top-left':
        return { top: rect.top - OFFSET, left: rect.left - OFFSET };
      case 'top-right':
        return { top: rect.top - OFFSET, left: rect.right + OFFSET };
      case 'bottom':
        return { top: rect.bottom + OFFSET, left: rect.left + rect.width / 2 };
      case 'bottom-left':
        return { top: rect.bottom + OFFSET, left: rect.left - OFFSET };
      case 'bottom-right':
        return { top: rect.bottom + OFFSET, left: rect.right + OFFSET };
      case 'left':
        return { top: rect.top + rect.height / 2, left: rect.left - OFFSET };
      case 'right':
        return { top: rect.top + rect.height / 2, left: rect.right + OFFSET };
    }
  }

  function trackMouse(event: MouseEvent) {
    mouse = { top: event.clientY, left: event.clientX };
  }

  function show(event: MouseEvent | FocusEvent) {
    if (disabled) return;

    if (event instanceof MouseEvent) {
      trackMouse(event);
    } else if (reference === 'mouse') {
      // Keyboard focus has no pointer, anchor to the trigger's center
      const rect = trigger.getBoundingClientRect();
      mouse = { top: rect.top + rect.height / 2, left: rect.left + rect.width / 2 };
    }

    clearTimeout(timeout);
    timeout = setTimeout(() => {
      position = calculatePosition(referenceBox());
      visible = true;
    }, delay);
  }

  function move(event: MouseEvent) {
    trackMouse(event);

    if (visible && reference === 'mouse') {
      position = calculatePosition(referenceBox());
    }
  }

  function hide() {
    clearTimeout(timeout);
    visible = false;
  }

  $effect(() => {
    if (disabled) hide();
  });

  // Clean up the pending timeout on unmount
  $effect(() => () => clearTimeout(timeout));

  const transforms: Record<TooltipPlacement, string> = {
    top: '-translate-x-1/2 -translate-y-full',
    bottom: '-translate-x-1/2',
    left: '-translate-x-full -translate-y-1/2',
    right: '-translate-y-1/2',
    'top-left': '-translate-x-full -translate-y-full',
    'bottom-left': '-translate-x-full',
    'top-right': '-translate-y-full',
    'bottom-right': ''
  };

  const wrapperClass = $derived(cn('relative inline-block', triggerClass));

  const finalTooltipClass = $derived(
    cn(
      'fixed z-1000 w-max max-w-32 px-3 py-2 rounded shadow',
      'bg-main border border-sidebar text-white text-xs font-normal text-left',
      'whitespace-normal break-words pointer-events-none',
      tooltipClass,
      transforms[placement]
    )
  );
</script>

<div
  bind:this={trigger}
  class={wrapperClass}
  role="presentation"
  aria-describedby={visible ? id : undefined}
  onmouseenter={show}
  onmousemove={move}
  onmouseleave={hide}
  onfocusin={show}
  onfocusout={hide}
>
  {@render children()}
</div>

{#if visible && !disabled}
  <div
    use:portal
    {id}
    role="tooltip"
    class={finalTooltipClass}
    style:top="{position.top}px"
    style:left="{position.left}px"
  >
    {#if typeof content === 'string'}
      {content}
    {:else}
      {@render content()}
    {/if}
  </div>
{/if}
