// U6: finding a card type in the cards sidebar as the catalogue grows.

import type { WellKnown } from '$types/api';

type Searchable = Pick<
	WellKnown,
	'key' | 'name_en' | 'name_es' | 'description_en' | 'description_es'
>;

/** Lowercase, without accents, so "dolar" finds "Dólar". */
export function fold(text: string): string {
	return text.normalize('NFD').replace(/[̀-ͯ]/g, '').toLowerCase();
}

/**
 * Whether a card type matches what was typed: every word appears in its name
 * (in either language), its description or its key. Nothing typed matches all.
 */
export function matchesCard(card: Searchable, query: string): boolean {
	const words = fold(query).split(/\s+/).filter(Boolean);
	if (words.length === 0) return true;
	const haystack = fold(
		[
			card.name_en,
			card.name_es,
			card.description_en,
			card.description_es,
			card.key.replace(/_/g, ' ')
		]
			.filter(Boolean)
			.join(' ')
	);
	return words.every((word) => haystack.includes(word));
}
