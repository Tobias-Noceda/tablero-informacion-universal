import { describe, expect, it } from 'vitest';
import type { SecretMeta, SecretScope } from '$types/api';
import { bindingsFor, refKey } from './binding';

const board = '6f1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d';

function meta(scope: SecretScope, name: string): SecretMeta {
	return {
		scope,
		name,
		kind: 'api_key',
		authorized: false,
		grants: [],
		created_at: '',
		updated_at: ''
	};
}

describe('bindingsFor', () => {
	it('references a board secret by name alone', () => {
		const { params, bindings } = bindingsFor(
			{ $credential: meta({ kind: 'board', owner: board }, 'KEY') },
			board
		);

		expect(params).toEqual({ $credential: '$KEY' });
		expect(bindings).toEqual({});
	});

	it('binds anything outside the board scope', () => {
		const mine = meta({ kind: 'member', owner: `${board}:me` }, 'MINE');
		const elsewhere = meta({ kind: 'board', owner: 'other-board' }, 'THEIRS');

		const { params, bindings } = bindingsFor({ $credential: mine, $other: elsewhere }, board);

		expect(params).toEqual({ $credential: '$MINE', $other: '$THEIRS' });
		expect(bindings).toEqual({
			MINE: { scope: mine.scope, name: 'MINE' },
			THEIRS: { scope: elsewhere.scope, name: 'THEIRS' }
		});
	});
});

describe('refKey', () => {
	it('is unique per scope and name', () => {
		const a = refKey(meta({ kind: 'user', owner: 'alice' }, 'KEY'));
		const b = refKey(meta({ kind: 'user', owner: 'bob' }, 'KEY'));
		const c = refKey(meta({ kind: 'board', owner: 'alice' }, 'KEY'));

		expect(new Set([a, b, c]).size).toBe(3);
	});
});
