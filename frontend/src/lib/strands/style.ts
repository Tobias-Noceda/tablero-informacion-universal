import type { Edge } from '@xyflow/svelte';
import type { Strand, StrandArrow, StrandColor, StrandStyle, StrandWidth } from '$types/api';

// What the backend accepts (models/strand.go), in the order the toolbar
// offers them.
export const STRAND_ARROWS: readonly StrandArrow[] = ['none', 'start', 'end', 'both'];
export const STRAND_WIDTHS: readonly StrandWidth[] = ['thin', 'normal', 'thick'];
export const STRAND_COLORS: readonly StrandColor[] = [
	'gray',
	'red',
	'orange',
	'yellow',
	'green',
	'blue',
	'purple',
	'pink'
];
export const MAX_STRAND_LABEL = 80;

// A strand on the canvas: its style plus whether it is the selected one.
export type StrandData = StrandStyle & { isSelected?: boolean };
export type StrandEdge = Edge<StrandData>;

export type ResolvedStrandStyle = Required<StrandStyle>;

const DEFAULT_STYLE: ResolvedStrandStyle = { label: '', arrow: 'none', width: 'normal', color: 'gray' };

const STROKE_WIDTHS: Record<StrandWidth, number> = { thin: 1, normal: 2, thick: 4 };

const STYLE_KEYS = ['label', 'arrow', 'width', 'color'] as const;

// Only the style fields that differ from the default, as the backend stores them.
export function styleOf(source: StrandStyle): StrandStyle {
	const style: StrandStyle = {};
	for (const key of STYLE_KEYS) {
		const value = source[key];
		if (value) Object.assign(style, { [key]: value });
	}
	return style;
}

export function resolveStyle(style: StrandStyle | undefined): ResolvedStrandStyle {
	return { ...DEFAULT_STYLE, ...styleOf(style ?? {}) };
}

export function strokeWidth(width: StrandWidth): number {
	return STROKE_WIDTHS[width];
}

// The palette lives in CSS (BoardContent/index.css) so each colour follows the theme.
export function strokeColor(color: StrandColor): string {
	return `var(--strand-${color})`;
}

export function arrowEnds(arrow: StrandArrow): { start: boolean; end: boolean } {
	return { start: arrow === 'start' || arrow === 'both', end: arrow === 'end' || arrow === 'both' };
}

export function strandToEdge(strand: Strand): StrandEdge {
	return { id: strand.id, source: strand.source, target: strand.target, data: styleOf(strand) };
}

// The edge with the patch applied; an empty value goes back to the default.
export function applyPatch(edge: StrandEdge, patch: StrandStyle): StrandEdge {
	const data: StrandData = { ...edge.data };
	for (const key of STYLE_KEYS) {
		if (!(key in patch)) continue;
		const value = patch[key];
		if (value) Object.assign(data, { [key]: value });
		else delete data[key];
	}
	return { ...edge, data };
}

// One line of at most MAX_STRAND_LABEL characters, as the backend keeps it.
export function normalizeLabel(label: string): string {
	return Array.from(label.replace(/[\p{Cc}]+/gu, ' ').trim())
		.slice(0, MAX_STRAND_LABEL)
		.join('')
		.trim();
}
