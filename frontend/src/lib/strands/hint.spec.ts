import { describe, expect, it } from 'vitest';

import { markStrandHintSeen, showStrandHint, strandHintKey, strandHintSeen } from './hint';

function memory() {
	const data = new Map<string, string>();
	return {
		data,
		getItem: (key: string) => data.get(key) ?? null,
		setItem: (key: string, value: string) => void data.set(key, value)
	};
}

describe('showStrandHint', () => {
	const board = { canEdit: true, cards: 2, strands: 0, seen: false };

	it('shows on an editable board with two cards and no strand', () => {
		expect(showStrandHint(board)).toBe(true);
		expect(showStrandHint({ ...board, cards: 5 })).toBe(true);
	});

	it('stays hidden otherwise', () => {
		expect(showStrandHint({ ...board, cards: 1 })).toBe(false);
		expect(showStrandHint({ ...board, strands: 1 })).toBe(false);
		expect(showStrandHint({ ...board, canEdit: false })).toBe(false);
		expect(showStrandHint({ ...board, seen: true })).toBe(false);
	});
});

describe('strandHintSeen', () => {
	it('is remembered per user', () => {
		const store = memory();
		expect(strandHintSeen('ana', store)).toBe(false);

		markStrandHintSeen('ana', store);
		expect(strandHintSeen('ana', store)).toBe(true);
		expect(strandHintSeen('bob', store)).toBe(false);
		expect(store.data.get(strandHintKey('ana'))).toBe('1');
	});

	it('counts as seen without a user, so nobody signed out gets it', () => {
		expect(strandHintSeen(null, memory())).toBe(true);
	});

	it('shows again when storage refuses, instead of failing', () => {
		const refusing = {
			getItem: () => {
				throw new Error('denied');
			},
			setItem: () => {
				throw new Error('denied');
			}
		};
		expect(strandHintSeen('ana', refusing)).toBe(false);
		expect(() => markStrandHintSeen('ana', refusing)).not.toThrow();
	});
});
