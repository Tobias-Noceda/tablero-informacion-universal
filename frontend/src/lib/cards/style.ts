import type { Node } from '@xyflow/svelte';
import type { CardBorder, CardBorderWidth, CardColor, CardStyle } from '$types/api';
import { STRAND_COLORS } from '$lib/strands/style';

// What the backend accepts (models/card_style.go), in the order the toolbar
// offers them.
export const CARD_BORDERS: readonly CardBorder[] = ['none', 'solid', 'dashed', 'dotted'];
export const CARD_BORDER_WIDTHS: readonly CardBorderWidth[] = ['thin', 'normal', 'thick'];
export const CARD_COLORS: readonly CardColor[] = [...STRAND_COLORS, 'white'];

export const MIN_CARD_WIDTH = 120;
export const MAX_CARD_WIDTH = 1600;
export const MIN_CARD_HEIGHT = 60;
export const MAX_CARD_HEIGHT = 1200;

const BORDER_PIXELS: Record<CardBorderWidth, number> = { thin: 1, normal: 2, thick: 4 };

const LOOK_KEYS = ['color', 'border', 'border_width', 'border_color'] as const;
const SIZE_KEYS = ['width', 'height'] as const;

// How a card looks apart from its size: the size is the canvas node's own
// width and height, so Svelte Flow lays the card out with it.
export type CardLook = Pick<CardStyle, (typeof LOOK_KEYS)[number]>;
export type CardSize = Pick<CardStyle, (typeof SIZE_KEYS)[number]>;

// Only the look fields that differ from the default, as the backend stores them.
export function lookOf(source: CardStyle | undefined): CardLook {
	const look: CardLook = {};
	for (const key of LOOK_KEYS) {
		const value = source?.[key];
		if (value) Object.assign(look, { [key]: value });
	}
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

// The card without its style fields, which cardNodeFields places instead.
export function withoutStyle<T extends CardStyle>(source: T): Omit<T, keyof CardStyle> {
	const rest: Partial<T> = { ...source };
	for (const key of [...LOOK_KEYS, ...SIZE_KEYS]) delete rest[key];
	return rest as Omit<T, keyof CardStyle>;
}

// The node with the patch applied; an empty value (0 for a size) goes back to the default.
export function applyCardPatch<N extends Node>(node: N, patch: CardStyle): N {
	const look: CardLook = { ...(node.data?.look as CardLook | undefined) };
	for (const key of LOOK_KEYS) {
		if (!(key in patch)) continue;
		const value = patch[key];
		if (value) Object.assign(look, { [key]: value });
		else delete look[key];
	}

	const next: N = { ...node, data: { ...node.data, look } };
	for (const key of SIZE_KEYS) {
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

// The palette lives in CSS (BoardContent/index.css). Fills are deep enough
// for the card's white text, except white, which takes dark text.
export function cardFill(color: CardColor | undefined): string | undefined {
	return color ? `var(--card-${color})` : undefined;
}

export function cardText(color: CardColor | undefined): string | undefined {
	return color === 'white' ? 'var(--card-text-on-white)' : undefined;
}

// A line colour follows the theme so it stands out from the canvas.
export function lineColor(color: CardColor): string {
	return color === 'white' ? 'var(--card-white)' : `var(--strand-${color})`;
}

// The border's own colour, or the card's when it has none. A white or
// uncoloured card has no tone of its own, so it takes the app's accent.
export function borderColorOf(look: CardLook): string {
	if (look.border_color) return lineColor(look.border_color);
	if (look.color && look.color !== 'white') return lineColor(look.color);
	return 'var(--card-accent)';
}

export function borderPixels(width: CardBorderWidth | undefined): number {
	return BORDER_PIXELS[width ?? 'normal'];
}

export function cardBorder(look: CardLook): string | undefined {
	if (!look.border || look.border === 'none') return undefined;
	return `${borderPixels(look.border_width)}px ${look.border} ${borderColorOf(look)}`;
}
