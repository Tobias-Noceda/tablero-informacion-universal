// U4: a one-off card saved as a card type (a post-it template) to reuse it.
// The template form opens filled in from the card, to be named and checked.

import type { Draft, ParameterRow, Row } from './draft';
import type { OneOffDraft, ValueRow } from './one-off';

/** Card types only send GET requests (the backend refuses anything else). */
export function canBecomeTemplate(card: Pick<OneOffDraft, 'method'>): boolean {
	return card.method === 'GET';
}

/** A name to start from: the card's title without its {{values}}. */
export function nameFrom(title: string): string {
	return title
		.replace(/\{\{\s*[^}]*\}\}/g, ' ')
		.replace(/\s+/g, ' ')
		.replace(/[\s:–-]+$/, '')
		.trim()
		.slice(0, 80);
}

/**
 * The template form filled in from the card. A credential the card uses
 * becomes a credential parameter ($credential, $credential_2, …), so whoever
 * uses the card type picks their own instead of reusing the card's.
 */
export function templateDraftFrom(card: OneOffDraft): Draft {
	const parameters: ParameterRow[] = [];
	const values = (list: ValueRow[]): Row[] =>
		list
			.filter((row) => row.key.trim())
			.map((row) => {
				if (!row.credential) return { key: row.key.trim(), value: row.value.trim() };
				const key =
					parameters.length === 0 ? '$credential' : `$credential_${parameters.length + 1}`;
				parameters.push({
					key,
					label: row.key.trim(),
					type: 'secret',
					placeholder: '',
					default: ''
				});
				return { key: row.key.trim(), value: key };
			});

	const headers = values(card.headers);
	const queries = values(card.queries);
	const outputs = card.outputs
		.filter((o) => o.key.trim() || o.query.trim())
		.map((o) => ({ key: o.key.trim(), label: o.key.trim(), query: o.query.trim(), example: '' }));

	return {
		name: nameFrom(card.title),
		description: '',
		resource: card.resource.trim(),
		response: card.response,
		rate: card.rate,
		headers,
		queries,
		parameters,
		outputs: outputs.length > 0 ? outputs : [{ key: '', label: '', query: '', example: '' }],
		title: card.title
	};
}
