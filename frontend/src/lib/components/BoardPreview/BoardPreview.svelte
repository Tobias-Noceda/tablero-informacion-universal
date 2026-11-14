<script lang="ts">
	import '$layouts/BoardContent/index.css';
	import type { Board } from '$types/api';
	import { m } from '$lib/paraglide/messages';
	import { borderColorOf, borderPixels, cardFill, cardText, lookOf } from '$lib/cards/style';
	import { resolveStyle, strokeColor, strokeWidth } from '$lib/strands/style';

	// A schematic of the board from what the board list already holds: where
	// the cards are, how they are strung, and the colours both were given on
	// the board. Cards are never executed here.
	let { board }: { board: Board } = $props();

	// Roughly a card on the canvas when it has no size of its own; positions
	// are its top-left corner.
	const CARD_W = 240;
	const CARD_H = 140;
	const PAD = 40;
	// The preview is scaled down, so lines are drawn thicker than on the canvas.
	const LINE_SCALE = 3;

	const cards = $derived(
		(board.postits ?? []).map((c) => {
			const look = lookOf(c);
			const bordered = !!look.border && look.border !== 'none';
			const borderWidth = borderPixels(look.border_width) * LINE_SCALE;
			return {
				id: c.id,
				x: c.position.x,
				y: c.position.y,
				w: c.width || CARD_W,
				h: c.height || CARD_H,
				fill: cardFill(look.color),
				text: cardText(look.color),
				stroke: bordered ? borderColorOf(look) : undefined,
				strokeWidth: bordered ? borderWidth : 4,
				dash:
					look.border === 'dashed'
						? `${borderWidth * 3} ${borderWidth * 2}`
						: look.border === 'dotted'
							? `${borderWidth} ${borderWidth}`
							: undefined
			};
		})
	);

	const viewBox = $derived.by(() => {
		if (cards.length === 0) return '0 0 1 1';
		const minX = Math.min(...cards.map((c) => c.x)) - PAD;
		const minY = Math.min(...cards.map((c) => c.y)) - PAD;
		const width = Math.max(...cards.map((c) => c.x + c.w)) + PAD - minX;
		const height = Math.max(...cards.map((c) => c.y + c.h)) + PAD - minY;
		return `${minX} ${minY} ${width} ${height}`;
	});

	const lines = $derived.by(() => {
		const byId = new Map(cards.map((c) => [c.id, c]));
		return (board.strands ?? []).flatMap((strand) => {
			const from = byId.get(strand.source);
			const to = byId.get(strand.target);
			if (!from || !to) return [];
			const style = resolveStyle(strand);
			return [
				{
					key: strand.id ?? `${strand.source}-${strand.target}`,
					x1: from.x + from.w / 2,
					y1: from.y + from.h / 2,
					x2: to.x + to.w / 2,
					y2: to.y + to.h / 2,
					color: strokeColor(style.color),
					width: strokeWidth(style.width) * LINE_SCALE
				}
			];
		});
	});
</script>

{#if cards.length === 0}
	<div class="flex h-full w-full items-center justify-center text-sm opacity-50">—</div>
{:else}
	<svg class="h-full w-full" {viewBox} preserveAspectRatio="xMidYMid meet" aria-hidden="true">
		{#each lines as line (line.key)}
			<line
				x1={line.x1}
				y1={line.y1}
				x2={line.x2}
				y2={line.y2}
				style:stroke={line.color}
				stroke-width={line.width}
			/>
		{/each}
		{#each cards as card (card.id)}
			<rect
				x={card.x}
				y={card.y}
				width={card.w}
				height={card.h}
				rx="16"
				class="fill-main stroke-main-border"
				style:fill={card.fill}
				style:stroke={card.stroke}
				stroke-width={card.strokeWidth}
				stroke-dasharray={card.dash}
			/>
			<text
				x={card.x + card.w / 2}
				y={card.y + card.h / 2}
				text-anchor="middle"
				dominant-baseline="middle"
				font-size="26"
				font-weight="700"
				class="fill-main-text"
				style:fill={card.text}
			>
				{`{{${m['landing.title_placeholder']()}}}`}
			</text>
		{/each}
	</svg>
{/if}
