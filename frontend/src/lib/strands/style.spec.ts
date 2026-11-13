import { describe, expect, it } from 'vitest';
import {
	applyPatch,
	arrowEnds,
	MAX_STRAND_LABEL,
	normalizeLabel,
	resolveStyle,
	strandToEdge,
	styleOf,
	strokeColor,
	strokeWidth
} from './style';

describe('strandToEdge', () => {
	it('keeps the link and the style', () => {
		expect(
			strandToEdge({ id: 's', source: 'a', target: 'b', label: 'feeds', arrow: 'end', color: 'blue' })
		).toEqual({ id: 's', source: 'a', target: 'b', data: { label: 'feeds', arrow: 'end', color: 'blue' } });
	});

	it('gives a strand saved before styles an empty style', () => {
		expect(strandToEdge({ id: 's', source: 'a', target: 'b' }).data).toEqual({});
	});
});

describe('resolveStyle', () => {
	it('fills in the defaults', () => {
		expect(resolveStyle(undefined)).toEqual({ label: '', arrow: 'none', width: 'normal', color: 'gray' });
		expect(resolveStyle({ width: 'thick' })).toEqual({ label: '', arrow: 'none', width: 'thick', color: 'gray' });
	});
});

describe('styleOf', () => {
	it('drops empty fields and anything that is not style', () => {
		expect(styleOf({ label: '', arrow: 'both', isSelected: true } as never)).toEqual({ arrow: 'both' });
	});
});

describe('applyPatch', () => {
	const edge = { id: 's', source: 'a', target: 'b', data: { label: 'feeds', color: 'red' as const, isSelected: true } };

	it('changes only what the patch names', () => {
		expect(applyPatch(edge, { arrow: 'start' }).data).toEqual({
			label: 'feeds',
			color: 'red',
			arrow: 'start',
			isSelected: true
		});
	});

	it('resets a field sent empty', () => {
		expect(applyPatch(edge, { label: '' }).data).toEqual({ color: 'red', isSelected: true });
	});

	it('leaves the original edge alone', () => {
		applyPatch(edge, { label: '' });
		expect(edge.data.label).toBe('feeds');
	});
});

describe('drawing', () => {
	it('maps each end of the arrow', () => {
		expect(arrowEnds('none')).toEqual({ start: false, end: false });
		expect(arrowEnds('start')).toEqual({ start: true, end: false });
		expect(arrowEnds('end')).toEqual({ start: false, end: true });
		expect(arrowEnds('both')).toEqual({ start: true, end: true });
	});

	it('grows with the width and reads the colour from the theme', () => {
		expect(strokeWidth('thin')).toBeLessThan(strokeWidth('normal'));
		expect(strokeWidth('normal')).toBeLessThan(strokeWidth('thick'));
		expect(strokeColor('blue')).toBe('var(--strand-blue)');
	});
});

describe('normalizeLabel', () => {
	it('trims and keeps one line', () => {
		expect(normalizeLabel('  depends\non  ')).toBe('depends on');
	});

	it('counts characters, not code units', () => {
		const label = normalizeLabel('ñ'.repeat(MAX_STRAND_LABEL + 5));
		expect(Array.from(label)).toHaveLength(MAX_STRAND_LABEL);
	});
});
