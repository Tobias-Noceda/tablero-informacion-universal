<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { CARD_BORDERS, CARD_COLORS, cardFill, type CardLook } from '$lib/cards/style';
	import { strokeColor } from '$lib/strands/style';
	import { cn } from '$lib/utils';
	import type { CardBorder, CardColor, CardStyle } from '$types/api';

	let {
		look,
		sized,
		onchange
	}: {
		look: CardLook;
		/** Whether the card has a size of its own, so it can go back to fitting its content. */
		sized: boolean;
		onchange: (patch: CardStyle) => void;
	} = $props();

	const borderNames: Record<CardBorder, () => string> = {
		none: m['cards.border_none'],
		solid: m['cards.border_solid'],
		dashed: m['cards.border_dashed'],
		dotted: m['cards.border_dotted']
	};

	// The palette is the strands' one, so are the colour names.
	const colorNames: Record<CardColor, () => string> = {
		gray: m['strands.color_gray'],
		red: m['strands.color_red'],
		orange: m['strands.color_orange'],
		yellow: m['strands.color_yellow'],
		green: m['strands.color_green'],
		blue: m['strands.color_blue'],
		purple: m['strands.color_purple'],
		pink: m['strands.color_pink']
	};

	const border = $derived(look.border ?? 'none');

	const option = (pressed: boolean) =>
		cn(
			'flex h-7 w-7 items-center justify-center rounded-md transition-colors',
			'hover:bg-(--app-background-hover) focus-visible:outline-2 focus-visible:outline-(--app-main)',
			'disabled:pointer-events-none disabled:opacity-40',
			pressed && 'bg-(--app-background-hover)'
		);

	const swatch = (pressed: boolean) =>
		cn(
			'h-4 w-4 rounded-full',
			pressed && 'ring-2 ring-(--app-foreground) ring-offset-1 ring-offset-(--app-background)'
		);
</script>

<div
	role="toolbar"
	aria-label={m['cards.toolbar']()}
	tabindex="-1"
	class="nodrag nopan nowheel flex items-center gap-1 rounded-lg border border-(--app-border) bg-(--app-background) p-1 text-(--app-foreground) shadow-lg"
	onclick={(event) => event.stopPropagation()}
	onkeydown={(event) => event.stopPropagation()}
>
	<button
		type="button"
		class={option(!look.color)}
		title={m['cards.color_default']()}
		aria-label={m['cards.color_default']()}
		aria-pressed={!look.color}
		onclick={() => onchange({ color: '' as CardColor })}
	>
		<span class={swatch(!look.color)} style:background-color="var(--app-main)"></span>
	</button>

	{#each CARD_COLORS as color (color)}
		<button
			type="button"
			class={option(look.color === color)}
			title={colorNames[color]()}
			aria-label={colorNames[color]()}
			aria-pressed={look.color === color}
			onclick={() => onchange({ color })}
		>
			<span
				class={swatch(look.color === color)}
				style:background-color={cardFill(color)}
				style:box-shadow="inset 0 0 0 2px {strokeColor(color)}"
			></span>
		</button>
	{/each}

	<span class="mx-0.5 h-5 w-px bg-(--app-border)" aria-hidden="true"></span>

	{#each CARD_BORDERS as kind (kind)}
		<button
			type="button"
			class={option(border === kind)}
			title={borderNames[kind]()}
			aria-label={borderNames[kind]()}
			aria-pressed={border === kind}
			onclick={() => onchange({ border: kind === 'none' ? ('' as CardBorder) : kind })}
		>
			<span
				class="h-4 w-5 rounded-sm"
				style:border={kind === 'none' ? '1px solid transparent' : `2px ${kind} currentColor`}
				style:background-color={kind === 'none' ? 'var(--app-background-hover)' : undefined}
			></span>
		</button>
	{/each}

	<span class="mx-0.5 h-5 w-px bg-(--app-border)" aria-hidden="true"></span>

	<button
		type="button"
		class={option(false)}
		title={m['cards.auto_size']()}
		aria-label={m['cards.auto_size']()}
		disabled={!sized}
		onclick={() => onchange({ width: 0, height: 0 })}
	>
		<svg viewBox="0 0 16 16" class="h-4 w-4" fill="none" stroke="currentColor" stroke-width="1.5" aria-hidden="true">
			<path d="M2 6V2h4M14 6V2h-4M2 10v4h4M14 10v4h-4" stroke-linecap="round" stroke-linejoin="round" />
		</svg>
	</button>
</div>
