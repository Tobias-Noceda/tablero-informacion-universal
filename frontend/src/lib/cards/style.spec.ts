import { describe, expect, it } from 'vitest';
import type { Node } from '@xyflow/svelte';
import {
	applyCardPatch,
	cardBorder,
	cardFill,
	cardNodeFields,
	clampSide,
	lookOf,
	MAX_CARD_HEIGHT,
	MAX_CARD_WIDTH,
	MIN_CARD_HEIGHT,
	MIN_CARD_WIDTH,
	resizePatch,
	sizeOf
} from './style';

describe('cardNodeFields', () => {
	it('puts the size on the node and the look in its data', () => {
		expect(cardNodeFields({ color: 'blue', border: 'dashed', width: 300, height: 200 })).toEqual({
			width: 300,
			height: 200,
			look: { color: 'blue', border: 'dashed' }
		});
	});

	it('gives a card saved before styles no size and no look', () => {
		expect(cardNodeFields({})).toEqual({ look: {} });
	});

	it('keeps one side alone', () => {
		expect(sizeOf({ height: 150 })).toEqual({ height: 150 });
		expect(lookOf({ border: 'solid' })).toEqual({ border: 'solid' });
	});
});

describe('applyCardPatch', () => {
	const node: Node = {
		id: 'c',
		position: { x: 0, y: 0 },
		width: 320,
		height: 180,
		data: { look: { color: 'red', border: 'solid' }, isSelected: true }
	};

	it('changes only what the patch names', () => {
		const next = applyCardPatch(node, { border: 'dotted' });
		expect(next.data).toEqual({ look: { color: 'red', border: 'dotted' }, isSelected: true });
		expect([next.width, next.height]).toEqual([320, 180]);
	});

	it('resets a field sent empty and a side sent as 0', () => {
		const next = applyCardPatch(node, { color: undefined, border: '' as never, width: 0, height: 0 });
		expect(next.data.look).toEqual({});
		expect('width' in next || 'height' in next).toBe(false);
	});

	it('leaves the original node alone', () => {
		applyCardPatch(node, { color: 'green', width: 0 });
		expect(node.width).toBe(320);
		expect((node.data.look as { color: string }).color).toBe('red');
	});
});

describe('resizePatch', () => {
	it('rounds to whole pixels', () => {
		expect(resizePatch(240.6, 99.4)).toEqual({ width: 241, height: 99 });
	});

	it('stays within the limits the backend keeps', () => {
		expect(resizePatch(10, 10)).toEqual({ width: MIN_CARD_WIDTH, height: MIN_CARD_HEIGHT });
		expect(resizePatch(9999, 9999)).toEqual({ width: MAX_CARD_WIDTH, height: MAX_CARD_HEIGHT });
		expect(clampSide(500, 100, 400)).toBe(400);
	});
});

describe('drawing', () => {
	it('reads the fill from the palette', () => {
		expect(cardFill('blue')).toBe('var(--card-blue)');
		expect(cardFill(undefined)).toBeUndefined();
	});

	it('draws a border only when one is chosen, in the card colour or gray', () => {
		expect(cardBorder({})).toBeUndefined();
		expect(cardBorder({ border: 'none', color: 'red' })).toBeUndefined();
		expect(cardBorder({ border: 'dashed', color: 'red' })).toBe('2px dashed var(--strand-red)');
		expect(cardBorder({ border: 'solid' })).toBe('2px solid var(--strand-gray)');
	});
});
