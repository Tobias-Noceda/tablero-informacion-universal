<script lang="ts">
	import type { Board } from '$types/api';
	import { m } from '$lib/paraglide/messages';

	// A schematic of the board from what the board list already holds: where
	// the cards are and how they are strung. Cards are never executed here.
	let { board }: { board: Board } = $props();

	// Roughly a card on the canvas; cards size themselves, positions are their top-left corner.
	const CARD_W = 240;
	const CARD_H = 140;
	const PAD = 40;

	const cards = $derived(board.postits ?? []);

	const viewBox = $derived.by(() => {
		if (cards.length === 0) return '0 0 1 1';
		const xs = cards.map((c) => c.position.x);
		const ys = cards.map((c) => c.position.y);
		const minX = Math.min(...xs) - PAD;
		const minY = Math.min(...ys) - PAD;
		const width = Math.max(...xs) + CARD_W + PAD - minX;
		const height = Math.max(...ys) + CARD_H + PAD - minY;
		return `${minX} ${minY} ${width} ${height}`;
	});

	const lines = $derived.by(() => {
		const byId = new Map(cards.map((c) => [c.id, c.position]));
		return (board.strands ?? []).flatMap((strand) => {
			const from = byId.get(strand.source);
			const to = byId.get(strand.target);
			if (!from || !to) return [];
			return [
				{
					key: `${strand.source}-${strand.target}`,
					x1: from.x + CARD_W / 2,
					y1: from.y + CARD_H / 2,
					x2: to.x + CARD_W / 2,
					y2: to.y + CARD_H / 2
				}
			];
		});
	});

	// A title with variables needs the card's data, so it is not shown.
	// function label(title: Board['postits'][number]['title']) {
	// 	if (!title?.text || title.vars) return '';
	// 	return title.text.length > 18 ? `${title.text.slice(0, 17)}…` : title.text;
	// }
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
				class="stroke-[#3e3e3e]"
				stroke-width="6"
			/>
		{/each}
		{#each cards as card (card.id)}
			<rect
				x={card.position.x}
				y={card.position.y}
				width={CARD_W}
				height={CARD_H}
				rx="16"
				class="fill-main stroke-main-border"
				stroke-width="4"
			/>
			<text
				x={card.position.x + CARD_W / 2}
				y={card.position.y + CARD_H / 2}
				text-anchor="middle"
				dominant-baseline="middle"
				font-size="26"
				font-weight="700"
				class="fill-main-text"
			>
				{`{{${m['landing.title_placeholder']()}}}`}
			</text>
		{/each}
	</svg>
{/if}
