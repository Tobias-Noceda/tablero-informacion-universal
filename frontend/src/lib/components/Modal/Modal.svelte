<script lang="ts">
  import Button from '$components/Button/Button.svelte';
  import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils';
	import type { Snippet } from 'svelte';

  interface Props {
    children: Snippet;
    onclose: () => void;
    acceptText?: string;
    acceptClass?: string;
    acceptDisabled?: boolean;
    onaccept: () => void;
    class?: string;
  }

  let { children, onclose, acceptText, acceptClass, acceptDisabled, onaccept, class: modalClass }: Props = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div
  class={cn(
    "z-200",
    modalClass,
    "fixed inset-0 flex items-center justify-center bg-black/50"
  )}
  role="presentation"
  onclick={onclose}
>
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div
    class="bg-background border border-main-border rounded-lg p-6 w-full max-w-md shadow-xl flex flex-col gap-4"
    role="dialog"
    aria-modal="true"
    onclick={(e) => e.stopPropagation()}
    tabindex={100}
  >
    {@render children()}
    <div class="flex gap-3 justify-end">
      <Button variant="gray" onclick={onclose}>{m.cancel()}</Button>
      <Button class={acceptClass} variant="success" onclick={onaccept} disabled={acceptDisabled}>{acceptText || m.accept()}</Button>
    </div>
  </div>
</div>
