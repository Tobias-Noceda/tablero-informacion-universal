import { describe, expect, it } from 'vitest';
import { changedCards } from './revisions';

describe('changedCards', () => {
	it('names the cards whose settings changed', () => {
		const before = [{ id: 'a' }, { id: 'b', rev: 1 }, { id: 'c', rev: 2 }];
		const after = [
			{ id: 'a', rev: 1 },
			{ id: 'b', rev: 1 },
			{ id: 'c', rev: 3 }
		];

		expect(changedCards(before, after)).toEqual(['a', 'c']);
	});

	it('leaves out cards that are new or gone', () => {
		expect(changedCards([{ id: 'gone', rev: 1 }], [{ id: 'new', rev: 4 }])).toEqual([]);
	});

	it('treats a missing revision as the first one', () => {
		expect(changedCards([{ id: 'a' }], [{ id: 'a', rev: 0 }])).toEqual([]);
	});
});
