<script lang="ts">
	import Icon from '$components/Icon/Icon.svelte';
	import { m } from '$lib/paraglide/messages';
	import {
		STRAND_ARROWS,
		STRAND_COLORS,
		STRAND_WIDTHS,
		strokeColor,
		strokeWidth,
		type ResolvedStrandStyle
	} from '$lib/strands/style';
	import { cn } from '$lib/utils';
	import type { StrandArrow, StrandColor, StrandStyle, StrandWidth } from '$types/api';

	let {
		style,
		onchange,
		onlabel,
		onremove
	}: {
		style: ResolvedStrandStyle;
		onchange: (patch: StrandStyle) => void;
		onlabel: () => void;
		onremove: () => void;
	} = $props();

	const arrowNames: Record<StrandArrow, () => string> = {
		none: m['strands.arrow_none'],
		start: m['strands.arrow_start'],
		end: m['strands.arrow_end'],
		both: m['strands.arrow_both']
	};

	const widthNames: Record<StrandWidth, () => string> = {
		thin: m['strands.width_thin'],
		normal: m['strands.width_normal'],
		thick: m['strands.width_thick']
	};

	const colorNames: Record<StrandColor, () => string> = {
		gray: m['strands.color_gray'],
		red: m['strands.color_red'],
		orange: m['strands.color_orange'],
		yellow: m['strands.color_yellow'],
		green: m['strands.color_green'],
		blue: m['strands.color_blue'],
		purple: m['strands.color_purple'],
		pink: m['strands.color_pink']
	};

	const option = (pressed: boolean) =>
		cn(
			'flex h-7 w-7 items-center justify-center rounded-md transition-colors',
			'hover:bg-(--app-background-hover) focus-visible:outline-2 focus-visible:outline-(--app-main)',
			pressed && 'bg-(--app-background-hover)'
		);
</script>

{#snippet arrowIcon(arrow: StrandArrow)}
	<svg viewBox="0 0 24 12" class="h-3 w-6" aria-hidden="true">
		<line x1="3" y1="6" x2="21" y2="6" stroke="currentColor" stroke-width="1.5" />
		{#if arrow === 'start' || arrow === 'both'}
			<path d="M2 6 L8 2.5 L8 9.5 Z" fill="currentColor" />
		{/if}
		{#if arrow === 'end' || arrow === 'both'}
			<path d="M22 6 L16 2.5 L16 9.5 Z" fill="currentColor" />
		{/if}
	</svg>
{/snippet}

<div
	role="toolbar"
	aria-label={m['strands.toolbar']()}
	tabindex="-1"
	class="nodrag nopan nowheel pointer-events-auto flex items-center gap-1 rounded-lg border border-(--app-border) bg-(--app-background) p-1 text-(--app-foreground) shadow-lg"
	onclick={(event) => event.stopPropagation()}
	onkeydown={(event) => event.stopPropagation()}
>
	<button type="button" class={option(false)} title={m['strands.label']()} aria-label={m['strands.label']()} onclick={onlabel}>
		<Icon name="edit" class="h-4 w-4" />
	</button>

	<span class="mx-0.5 h-5 w-px bg-(--app-border)" aria-hidden="true"></span>

	{#each STRAND_ARROWS as arrow (arrow)}
		<button
			type="button"
			class={option(style.arrow === arrow)}
			title={arrowNames[arrow]()}
			aria-label={arrowNames[arrow]()}
			aria-pressed={style.arrow === arrow}
			onclick={() => onchange({ arrow })}
		>
			{@render arrowIcon(arrow)}
		</button>
	{/each}

	<span class="mx-0.5 h-5 w-px bg-(--app-border)" aria-hidden="true"></span>

	{#each STRAND_WIDTHS as width (width)}
		<button
			type="button"
			class={option(style.width === width)}
			title={widthNames[width]()}
			aria-label={widthNames[width]()}
			aria-pressed={style.width === width}
			onclick={() => onchange({ width })}
		>
			<svg viewBox="0 0 20 10" class="h-3 w-5" aria-hidden="true">
				<line x1="2" y1="5" x2="18" y2="5" stroke="currentColor" stroke-width={strokeWidth(width)} stroke-linecap="round" />
			</svg>
		</button>
	{/each}

	<span class="mx-0.5 h-5 w-px bg-(--app-border)" aria-hidden="true"></span>

	{#each STRAND_COLORS as color (color)}
		<button
			type="button"
			class={option(style.color === color)}
			title={colorNames[color]()}
			aria-label={colorNames[color]()}
			aria-pressed={style.color === color}
			onclick={() => onchange({ color })}
		>
			<span
				class={cn('h-4 w-4 rounded-full', style.color === color && 'ring-2 ring-(--app-foreground) ring-offset-1 ring-offset-(--app-background)')}
				style:background-color={strokeColor(color)}
			></span>
		</button>
	{/each}

	<span class="mx-0.5 h-5 w-px bg-(--app-border)" aria-hidden="true"></span>

	<button
		type="button"
		class={cn(option(false), 'text-(--app-color-destructive)')}
		title={m['strands.remove']()}
		aria-label={m['strands.remove']()}
		onclick={onremove}
	>
		<Icon name="trash" class="h-4 w-4" />
	</button>
</div>
