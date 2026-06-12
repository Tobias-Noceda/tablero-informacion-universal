<script lang="ts">
  import Button from '$components/Button/Button.svelte';
  import { m } from '$lib/paraglide/messages';
	import type { Snippet } from 'svelte';

  interface Props {
    children: Snippet;
    onclose: () => void;
    acceptText?: string;
    acceptDisabled?: boolean;
    onaccept: () => void;
  }

  let { children, onclose, acceptText, acceptDisabled, onaccept }: Props = $props();

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') onclose();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

<div
  class="fixed inset-0 z-50 flex items-center justify-center bg-black/50"
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
      <Button variant="secondary" onclick={onclose}>{m.cancel()}</Button>
      <Button variant="success" onclick={onaccept} disabled={acceptDisabled}>{acceptText || m.accept()}</Button>
    </div>
  </div>
</div>
