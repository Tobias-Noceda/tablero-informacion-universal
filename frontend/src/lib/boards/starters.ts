// U5: example boards to start from, so a new board is not a blank canvas.
// Built from card types that need no credential, placed and linked, and made
// with the same API calls as by hand: create each card, move it, colour it,
// then draw the strands.

import type { CardColor } from '$types/api';

export type StarterKey = 'finance' | 'trip' | 'dev';

export type StarterCard = {
	/** How the starter's strands name this card. */
	ref: string;
	wellKnown: string;
	/** A note's heading and text, as message keys, instead of the type's title. */
	note?: { title: string; text: string };
	params?: (today: Date) => Record<string, string>;
	at: { x: number; y: number };
	color?: CardColor;
};

export type Starter = { key: StarterKey; cards: StarterCard[]; strands: [string, string][] };

const day = (from: Date, plus: number) => {
	const d = new Date(from.getFullYear(), from.getMonth(), from.getDate() + plus);
	return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;
};

export const STARTERS: Starter[] = [
	{
		key: 'finance',
		cards: [
			{ ref: 'dollar', wellKnown: 'dolar_oficial', at: { x: 0, y: 0 }, color: 'green' },
			{ ref: 'risk', wellKnown: 'riesgo_pais', at: { x: 0, y: 180 }, color: 'orange' },
			{
				ref: 'btc',
				wellKnown: 'crypto_price',
				params: () => ({ $coin: 'bitcoin', $currency: 'usd' }),
				at: { x: 420, y: 0 },
				color: 'yellow'
			},
			{
				ref: 'eth',
				wellKnown: 'crypto_price',
				params: () => ({ $coin: 'ethereum', $currency: 'usd' }),
				at: { x: 420, y: 180 },
				color: 'purple'
			},
			{
				ref: 'note',
				wellKnown: 'static_card',
				note: { title: 'starters.finance_note_title', text: 'starters.finance_note' },
				at: { x: 210, y: 380 }
			}
		],
		strands: [
			['note', 'dollar'],
			['note', 'btc']
		]
	},
	{
		key: 'trip',
		cards: [
			{
				ref: 'weather',
				wellKnown: 'temperature',
				params: (today) => ({ $start_date: day(today, 0), $end_date: day(today, 2) }),
				at: { x: 0, y: 0 },
				color: 'blue'
			},
			{ ref: 'air', wellKnown: 'air_quality', at: { x: 0, y: 200 }, color: 'green' },
			{ ref: 'dollar', wellKnown: 'dolar_oficial', at: { x: 420, y: 0 } },
			{
				ref: 'pack',
				wellKnown: 'static_card',
				note: { title: 'starters.trip_note_title', text: 'starters.trip_note' },
				at: { x: 420, y: 200 },
				color: 'yellow'
			}
		],
		strands: [
			['weather', 'pack'],
			['dollar', 'pack']
		]
	},
	{
		key: 'dev',
		cards: [
			{
				ref: 'goals',
				wellKnown: 'static_card',
				note: { title: 'starters.dev_note_title', text: 'starters.dev_note' },
				at: { x: 210, y: 0 },
				color: 'purple'
			},
			{
				ref: 'svelte',
				wellKnown: 'github_repo',
				params: () => ({ $query: 'repo:sveltejs/svelte' }),
				at: { x: 0, y: 220 }
			},
			{
				ref: 'go',
				wellKnown: 'github_repo',
				params: () => ({ $query: 'repo:golang/go' }),
				at: { x: 420, y: 220 }
			},
			{ ref: 'break', wellKnown: 'dog_facts', at: { x: 210, y: 420 }, color: 'pink' }
		],
		strands: [
			['goals', 'svelte'],
			['goals', 'go']
		]
	}
];

/** What filling a board needs from the API (the post-it and strand services). */
export type StarterApi = {
	create(
		wellKnown: string,
		title: { text: string; vars: boolean },
		params: Record<string, string>
	): Promise<{ id: string }>;
	move(id: string, x: number, y: number): Promise<unknown>;
	style(id: string, color: CardColor): Promise<unknown>;
	connect(source: string, target: string): Promise<unknown>;
};

/**
 * Puts the starter's cards and strands on a new board. A card that fails
 * (a plan limit, a source down) is skipped with its strands; the count of
 * what failed is answered so the board can say the example is incomplete.
 */
export async function fillStarter(
	starter: Starter,
	api: StarterApi,
	titleOf: (card: StarterCard) => { text: string; vars: boolean },
	textOf: (key: string) => string,
	today = new Date()
): Promise<{ failed: number }> {
	const ids = new Map<string, string>();
	let failed = 0;

	for (const card of starter.cards) {
		try {
			const params = card.note ? { text: textOf(card.note.text) } : (card.params?.(today) ?? {});
			const { id } = await api.create(card.wellKnown, titleOf(card), params);
			ids.set(card.ref, id);
			await api.move(id, card.at.x, card.at.y);
			if (card.color) await api.style(id, card.color);
		} catch {
			failed++;
		}
	}

	for (const [from, to] of starter.strands) {
		const source = ids.get(from);
		const target = ids.get(to);
		if (!source || !target) continue;
		try {
			await api.connect(source, target);
		} catch {
			failed++;
		}
	}

	return { failed };
}
