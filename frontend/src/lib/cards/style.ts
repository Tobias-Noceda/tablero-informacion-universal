import type { Node } from '@xyflow/svelte';
import type { CardBorder, CardColor, CardStyle } from '$types/api';
import { STRAND_COLORS } from '$lib/strands/style';

// What the backend accepts (models/card_style.go), in the order the toolbar
// offers them.
export const CARD_BORDERS: readonly CardBorder[] = ['none', 'solid', 'dashed', 'dotted'];
export const CARD_COLORS: readonly CardColor[] = STRAND_COLORS;

export const MIN_CARD_WIDTH = 120;
export const MAX_CARD_WIDTH = 1600;
export const MIN_CARD_HEIGHT = 60;
export const MAX_CARD_HEIGHT = 1200;

// How a card looks apart from its size: the size is the canvas node's own
// width and height, so Svelte Flow lays the card out with it.
export type CardLook = Pick<CardStyle, 'color' | 'border'>;
export type CardSize = Pick<CardStyle, 'width' | 'height'>;

// Only the colour and border that differ from the default, as the backend stores them.
export function lookOf(source: CardStyle | undefined): CardLook {
	const look: CardLook = {};
	if (source?.color) look.color = source.color;
	if (source?.border) look.border = source.border;
	return look;
}

// Only the sides that are set; an unset side follows the content.
export function sizeOf(source: CardStyle | undefined): CardSize {
	const size: CardSize = {};
	if (source?.width) size.width = source.width;
	if (source?.height) size.height = source.height;
	return size;
}

// A board card as a canvas node: its size on the node, its look in its data.
export function cardNodeFields(source: CardStyle): CardSize & { look: CardLook } {
	return { ...sizeOf(source), look: lookOf(source) };
}

// The node with the patch applied; an empty value (0 for a size) goes back to the default.
export function applyCardPatch<N extends Node>(node: N, patch: CardStyle): N {
	const look: CardLook = { ...(node.data?.look as CardLook | undefined) };
	for (const key of ['color', 'border'] as const) {
		if (!(key in patch)) continue;
		const value = patch[key];
		if (value) Object.assign(look, { [key]: value });
		else delete look[key];
	}

	const next: N = { ...node, data: { ...node.data, look } };
	for (const key of ['width', 'height'] as const) {
		if (!(key in patch)) continue;
		const value = patch[key];
		if (value) next[key] = value;
		else delete next[key];
	}
	return next;
}

// A side rounded to whole pixels within the limits the backend keeps.
export function clampSide(value: number, min: number, max: number): number {
	return Math.min(max, Math.max(min, Math.round(value)));
}

// What a resize saves: both sides, within the limits.
export function resizePatch(width: number, height: number): Required<CardSize> {
	return {
		width: clampSide(width, MIN_CARD_WIDTH, MAX_CARD_WIDTH),
		height: clampSide(height, MIN_CARD_HEIGHT, MAX_CARD_HEIGHT)
	};
}

// The palette lives in CSS (BoardContent/index.css): the fill keeps the card's
// white text legible, the border follows the theme so it stands out from the canvas.
export function cardFill(color: CardColor | undefined): string | undefined {
	return color ? `var(--card-${color})` : undefined;
}

export function cardBorder(look: CardLook): string | undefined {
	if (!look.border || look.border === 'none') return undefined;
	return `2px ${look.border} var(--strand-${look.color ?? 'gray'})`;
}
