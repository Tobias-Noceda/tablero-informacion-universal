<script lang="ts">
	import type { Snippet } from 'svelte';
	import { fade, slide } from 'svelte/transition';
	import { cn } from '$lib/utils';

	interface Props {
		open: boolean;
		onclose: () => void;
		children: Snippet;
		/** Distance from the top of the viewport in px, e.g. the height of the bar it drops from. */
		top?: number;
		/** Accessible name of the panel. */
		label?: string;
		/** Classes for the panel itself. */
		class?: string;
		/** Rendered centred under the panel, sliding with it, e.g. a tab that closes it. */
		handle?: Snippet;
	}

	let { open, onclose, children, top = 0, label, class: panelClass, handle }: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (open && e.key === 'Escape') onclose();
	}
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- A panel that drops down from `top`; tapping the dimmed rest closes it. -->
{#if open}
	<div class="fixed inset-x-0 bottom-0 z-150" style:top="{top}px">
		<div
			class="absolute inset-0 bg-black/40"
			role="presentation"
			onclick={onclose}
			transition:fade={{ duration: 150 }}
		></div>
		<div class="relative flex flex-col items-center" transition:slide={{ duration: 150 }}>
			<div
				role="dialog"
				aria-modal="true"
				aria-label={label}
				class={cn(
					'flex w-full flex-col gap-3 border-b border-main-border bg-main p-4 shadow-lg',
					panelClass
				)}
			>
				{@render children()}
			</div>
			{#if handle}
				{@render handle()}
			{/if}
		</div>
	</div>
{/if}
