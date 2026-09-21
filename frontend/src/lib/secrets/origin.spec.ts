import { describe, expect, it } from 'vitest';
import type { SecretMeta, SecretScope } from '$types/api';
import { groupByOrigin, originOf } from './origin';

const board = '6f1b2c3d-4e5f-4a6b-8c7d-9e0f1a2b3c4d';
const otherBoard = '11111111-2222-4333-8444-555555555555';
const me = 'me';

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

describe('originOf', () => {
	it('tells where a usable secret comes from', () => {
		const cases: [SecretScope, ReturnType<typeof originOf>][] = [
			[{ kind: 'board', owner: board }, 'board'],
			[{ kind: 'member', owner: `${board}:${me}` }, 'mine'],
			[{ kind: 'user', owner: me }, 'profile'],
			[{ kind: 'group', owner: otherBoard }, 'group'],
			[{ kind: 'user', owner: 'someone-else' }, 'shared'],
			[{ kind: 'board', owner: otherBoard }, 'shared'],
			[{ kind: 'member', owner: `${board}:someone-else` }, 'shared']
		];

		for (const [scope, want] of cases) {
			expect(originOf(meta(scope, 'X'), board, me), scope.kind + ':' + scope.owner).toBe(want);
		}
	});
});

describe('groupByOrigin', () => {
	it('keeps every origin in a fixed order, only when it has secrets', () => {
		const groups = groupByOrigin(
			[
				meta({ kind: 'user', owner: 'alice' }, 'SHARED_WITH_ME'),
				meta({ kind: 'board', owner: board }, 'BOARDS'),
				meta({ kind: 'member', owner: `${board}:${me}` }, 'MINE')
			],
			board,
			me
		);

		expect(groups.map((g) => g.origin)).toEqual(['board', 'mine', 'shared']);
		expect(groups[2].secrets.map((s) => s.name)).toEqual(['SHARED_WITH_ME']);
	});

	it('is empty when nothing is usable', () => {
		expect(groupByOrigin([], board, me)).toEqual([]);
	});
});
