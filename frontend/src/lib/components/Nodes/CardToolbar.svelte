<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import {
		borderColorOf,
		borderPixels,
		CARD_BORDER_WIDTHS,
		CARD_BORDERS,
		CARD_COLORS,
		cardFill,
		lineColor,
		type CardLook
	} from '$lib/cards/style';
	import { cn } from '$lib/utils';
	import type { CardBorder, CardBorderWidth, CardColor, CardStyle } from '$types/api';

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

	const widthNames: Record<CardBorderWidth, () => string> = {
		thin: m['cards.border_thin'],
		normal: m['cards.border_normal'],
		thick: m['cards.border_thick']
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
		pink: m['strands.color_pink'],
		white: m['cards.color_white']
	};

	// An empty value is how the backend resets a field.
	const reset = '' as never;

	const border = $derived(look.border ?? 'none');
	const hasBorder = $derived(border !== 'none');
	const borderWidth = $derived(look.border_width ?? 'normal');
	// What the border shows without a colour of its own.
	const followColor = $derived(borderColorOf({ color: look.color }));

	const option = (pressed: boolean) =>
		cn(
			'flex h-7 w-7 items-center justify-center rounded-md transition-colors',
			'hover:bg-(--app-background-hover) focus-visible:outline-2 focus-visible:outline-(--app-main)',
			'disabled:pointer-events-none disabled:opacity-40',
			pressed && 'bg-(--app-background-hover)'
		);

	const swatch = (pressed: boolean) =>
		cn(
			'h-4 w-4 rounded-full border border-(--app-border)',
			pressed && 'ring-2 ring-(--app-foreground) ring-offset-1 ring-offset-(--app-background)'
		);
</script>

{#snippet separator()}
	<span class="mx-0.5 h-5 w-px bg-(--app-border)" aria-hidden="true"></span>
{/snippet}

{#snippet rowLabel(text: string)}
	<span class="w-14 shrink-0 px-1 text-xs opacity-70">{text}</span>
{/snippet}

<div
	role="toolbar"
	aria-label={m['cards.toolbar']()}
	tabindex="-1"
	class="nodrag nopan nowheel flex flex-col gap-1 rounded-lg border border-(--app-border) bg-(--app-background) p-1 text-(--app-foreground) shadow-lg"
	onclick={(event) => event.stopPropagation()}
	onkeydown={(event) => event.stopPropagation()}
>
	<div class="flex items-center gap-1" role="group" aria-label={m['cards.fill']()}>
		{@render rowLabel(m['cards.fill']())}

		<button
			type="button"
			class={option(!look.color)}
			title={m['cards.color_default']()}
			aria-label={m['cards.color_default']()}
			aria-pressed={!look.color}
			onclick={() => onchange({ color: reset })}
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
				<span class={swatch(look.color === color)} style:background-color={cardFill(color)}></span>
			</button>
		{/each}

		{@render separator()}

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

	<div class="flex items-center gap-1" role="group" aria-label={m['cards.border']()}>
		{@render rowLabel(m['cards.border']())}

		{#each CARD_BORDERS as kind (kind)}
			<button
				type="button"
				class={option(border === kind)}
				title={borderNames[kind]()}
				aria-label={borderNames[kind]()}
				aria-pressed={border === kind}
				onclick={() => onchange({ border: kind === 'none' ? reset : kind })}
			>
				<span
					class="h-4 w-5 rounded-sm"
					style:border={kind === 'none' ? '1px solid transparent' : `2px ${kind} currentColor`}
					style:background-color={kind === 'none' ? 'var(--app-background-hover)' : undefined}
				></span>
			</button>
		{/each}

		{@render separator()}

		{#each CARD_BORDER_WIDTHS as width (width)}
			<button
				type="button"
				class={option(hasBorder && borderWidth === width)}
				title={widthNames[width]()}
				aria-label={widthNames[width]()}
				aria-pressed={hasBorder && borderWidth === width}
				disabled={!hasBorder}
				onclick={() => onchange({ border_width: width === 'normal' ? reset : width })}
			>
				<svg viewBox="0 0 20 10" class="h-3 w-5" aria-hidden="true">
					<line x1="2" y1="5" x2="18" y2="5" stroke="currentColor" stroke-width={borderPixels(width)} />
				</svg>
			</button>
		{/each}

		{@render separator()}

		<button
			type="button"
			class={option(hasBorder && !look.border_color)}
			title={m['cards.border_follow']()}
			aria-label={m['cards.border_follow']()}
			aria-pressed={hasBorder && !look.border_color}
			disabled={!hasBorder}
			onclick={() => onchange({ border_color: reset })}
		>
			<span
				class={cn(swatch(hasBorder && !look.border_color), 'bg-transparent')}
				style:box-shadow="inset 0 0 0 3px {followColor}"
			></span>
		</button>

		{#each CARD_COLORS as color (color)}
			<button
				type="button"
				class={option(look.border_color === color)}
				title={colorNames[color]()}
				aria-label={colorNames[color]()}
				aria-pressed={look.border_color === color}
				disabled={!hasBorder}
				onclick={() => onchange({ border_color: color })}
			>
				<span class={swatch(look.border_color === color)} style:background-color={lineColor(color)}></span>
			</button>
		{/each}
	</div>
</div>
