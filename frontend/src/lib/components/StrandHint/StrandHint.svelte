<script lang="ts">
	import { m } from '$lib/paraglide/messages';

	interface Props {
		ondismiss: () => void;
	}

	let { ondismiss }: Props = $props();
</script>

<!-- Shown over the board until the first strand is drawn or this is dismissed. -->
<div
	role="status"
	class="flex max-w-sm items-center gap-3 rounded-lg border border-main-border bg-background px-3 py-2 text-sm text-foreground shadow-md"
>
	<svg
		viewBox="0 0 64 28"
		class="h-7 w-16 shrink-0 text-main"
		aria-hidden="true"
		fill="none"
		stroke="currentColor"
		stroke-width="2"
	>
		<rect x="2" y="6" width="16" height="16" rx="3" />
		<rect x="46" y="6" width="16" height="16" rx="3" />
		<path d="M18 14 H44" stroke-dasharray="4 3" class="strand-hint-line" />
		<path d="M39 9 L45 14 L39 19" />
	</svg>
	<p class="flex-1">{m['strand_hint.text']()}</p>
	<button
		type="button"
		class="shrink-0 cursor-pointer rounded-md px-2 py-1 text-xs font-semibold hover:bg-background-hover"
		onclick={ondismiss}
	>
		{m['strand_hint.dismiss']()}
	</button>
</div>

<style>
	.strand-hint-line {
		animation: strand-hint-draw 1.6s linear infinite;
	}

	@keyframes strand-hint-draw {
		to {
			stroke-dashoffset: -14;
		}
	}

	@media (prefers-reduced-motion: reduce) {
		.strand-hint-line {
			animation: none;
		}
	}
</style>
