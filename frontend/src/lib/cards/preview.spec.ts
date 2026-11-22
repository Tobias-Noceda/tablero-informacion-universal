import { describe, expect, it } from 'vitest';

import { childrenOf, jqPath, lastIndex, leafText, outputName, TREE_CHILDREN } from './preview';

describe('jqPath', () => {
	it('reads the whole response at the root', () => {
		expect(jqPath([])).toBe('.');
	});

	it('chains keys and indices', () => {
		expect(jqPath(['data', 0, 'price'])).toBe('.data[0].price');
		expect(jqPath([0, 'name'])).toBe('.[0].name');
	});

	it('quotes keys that are not identifiers', () => {
		expect(jqPath(['main', 'feels like'])).toBe('.main["feels like"]');
		expect(jqPath(['2024'])).toBe('.["2024"]');
		expect(jqPath(['a"b'])).toBe('.["a\\"b"]');
	});

	it('widens one index to every element in a list', () => {
		expect(jqPath(['data', 0, 'price'], 1)).toBe('[.data[].price]');
		expect(jqPath([3], 0)).toBe('[.[]]');
	});
});

describe('lastIndex', () => {
	it('finds the innermost array index', () => {
		expect(lastIndex(['data', 0, 'items', 2, 'id'])).toBe(3);
		expect(lastIndex(['data', 'id'])).toBeUndefined();
	});
});

describe('outputName', () => {
	it('takes the last key', () => {
		expect(outputName(['data', 0, 'price'], [])).toBe('price');
	});

	it('makes it an identifier the title can use', () => {
		expect(outputName(['main', 'Feels Like'], [])).toBe('feels_like');
		expect(outputName(['Año'], [])).toBe('ano');
		expect(outputName(['2024'], [])).toBe('v_2024');
		expect(outputName([0], [])).toBe('value');
	});

	it('does not repeat a name already used', () => {
		expect(outputName(['price'], ['price'])).toBe('price_2');
		expect(outputName(['price'], ['price', 'price_2'])).toBe('price_3');
	});

	it('says when it is a list', () => {
		expect(outputName(['data', 0, 'price'], [], true)).toBe('price_list');
	});
});

describe('childrenOf', () => {
	it('lists object entries and array items', () => {
		expect(childrenOf({ a: 1, b: 2 }).entries).toEqual([
			['a', 1],
			['b', 2]
		]);
		expect(childrenOf(['x']).entries).toEqual([[0, 'x']]);
		expect(childrenOf('leaf')).toEqual({ entries: [], hidden: 0 });
		expect(childrenOf(null)).toEqual({ entries: [], hidden: 0 });
	});

	it('shows a long list in part', () => {
		const long = Array.from({ length: TREE_CHILDREN + 7 }, (_, i) => i);
		const { entries, hidden } = childrenOf(long);
		expect(entries).toHaveLength(TREE_CHILDREN);
		expect(hidden).toBe(7);
	});
});

describe('leafText', () => {
	it('shows JSON and cuts long text', () => {
		expect(leafText('hi')).toBe('"hi"');
		expect(leafText(3.5)).toBe('3.5');
		expect(leafText(null)).toBe('null');
		expect(leafText('x'.repeat(100), 10)).toBe('"xxxxxxxx…');
	});
});
