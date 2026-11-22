import { describe, expect, it } from 'vitest';

import { fillStarter, STARTERS, type StarterApi } from './starters';

// Card types that run without a credential or a platform key.
const NO_CREDENTIAL = new Set([
	'static_card',
	'temperature',
	'air_quality',
	'dolar_oficial',
	'riesgo_pais',
	'crypto_price',
	'github_repo',
	'dog_facts'
]);

describe('STARTERS', () => {
	it.each(STARTERS.map((s) => [s.key, s] as const))('%s is consistent', (_, starter) => {
		const refs = starter.cards.map((c) => c.ref);
		expect(new Set(refs).size).toBe(refs.length);
		for (const [from, to] of starter.strands) {
			expect(refs).toContain(from);
			expect(refs).toContain(to);
		}
		const spots = starter.cards.map((c) => `${c.at.x},${c.at.y}`);
		expect(new Set(spots).size).toBe(spots.length);
		// Well under the free plan's 25 cards per board.
		expect(starter.cards.length).toBeLessThanOrEqual(6);
		for (const card of starter.cards) expect(NO_CREDENTIAL.has(card.wellKnown)).toBe(true);
	});

	it('gives the trip its next days', () => {
		const weather = STARTERS.find((s) => s.key === 'trip')!.cards.find((c) => c.ref === 'weather')!;
		expect(weather.params!(new Date(2026, 11, 31))).toEqual({
			$start_date: '2026-12-31',
			$end_date: '2027-01-02'
		});
	});
});

function recorder(fail: (wellKnown: string) => boolean = () => false) {
	const calls: string[] = [];
	let next = 0;
	const api: StarterApi = {
		async create(wellKnown, title, params) {
			if (fail(wellKnown)) throw new Error('refused');
			const id = `id${next++}`;
			calls.push(`create ${wellKnown} "${title.text}" ${JSON.stringify(params)} -> ${id}`);
			return { id };
		},
		async move(id, x, y) {
			calls.push(`move ${id} ${x},${y}`);
		},
		async style(id, color) {
			calls.push(`style ${id} ${color}`);
		},
		async connect(source, target) {
			calls.push(`connect ${source} ${target}`);
		}
	};
	return { api, calls };
}

describe('fillStarter', () => {
	const dev = STARTERS.find((s) => s.key === 'dev')!;
	const titleOf = (card: { wellKnown: string; note?: { title: string } }) => ({
		text: card.note ? `T:${card.note.title}` : card.wellKnown,
		vars: false
	});
	const textOf = (key: string) => `X:${key}`;

	it('creates, places and colours every card, then links them', async () => {
		const { api, calls } = recorder();
		expect(await fillStarter(dev, api, titleOf, textOf)).toEqual({ failed: 0 });
		expect(calls).toEqual([
			'create static_card "T:starters.dev_note_title" {"text":"X:starters.dev_note"} -> id0',
			'move id0 210,0',
			'style id0 purple',
			'create github_repo "github_repo" {"$query":"repo:sveltejs/svelte"} -> id1',
			'move id1 0,220',
			'create github_repo "github_repo" {"$query":"repo:golang/go"} -> id2',
			'move id2 420,220',
			'create dog_facts "dog_facts" {} -> id3',
			'move id3 210,420',
			'style id3 pink',
			'connect id0 id1',
			'connect id0 id2'
		]);
	});

	it('skips a refused card and its strands, and says how many failed', async () => {
		const { api, calls } = recorder((wk) => wk === 'static_card');
		expect(await fillStarter(dev, api, titleOf, textOf)).toEqual({ failed: 1 });
		expect(calls.some((c) => c.startsWith('connect'))).toBe(false);
	});
});
