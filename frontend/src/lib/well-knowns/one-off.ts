import { bindingsFor, refKey } from '$lib/secrets/binding';
import { analyzeTitle } from '$lib/title';
import type { OneOffRequest, PostItSettings, SecretMeta, SecretRef, UUID } from '$types/api';

// A card with its own request and no card type (a "one-off" card). The board
// stores it with an empty type; the canvas knows it by this key.
export const ONE_OFF = 'one_off';

// The board's type for a card: one without a type is a one-off card.
export function nodeType(type: string | undefined | null) {
	return type || ONE_OFF;
}

// A query string parameter or header: a fixed value, or a picked credential
// (its refKey), sent as "$NAME".
export type ValueRow = { key: string; value: string; credential: boolean; secret: string };
export type OutputRow = { key: string; query: string };

export type OneOffDraft = {
	title: string;
	resource: string;
	response: 'json' | 'html';
	rate: number;
	queries: ValueRow[];
	headers: ValueRow[];
	outputs: OutputRow[];
};

export function emptyOneOff(): OneOffDraft {
	return {
		title: '',
		resource: '',
		response: 'json',
		rate: 600,
		queries: [],
		headers: [],
		outputs: [{ key: '', query: '' }]
	};
}

// A "$NAME" a card sends, as the secret it points to: its binding, else the
// board's own secret by that name.
const SECRET_REF = /^\$([A-Z][A-Z0-9_]*)$/;

function rowsOf(
	record: Record<string, string> | null | undefined,
	bindings: Record<string, SecretRef>,
	board: UUID
): ValueRow[] {
	return Object.entries(record ?? {}).map(([key, value]) => {
		const name = SECRET_REF.exec(value)?.[1];
		if (!name) return { key, value, credential: false, secret: '' };
		const ref = bindings[name] ?? { scope: { kind: 'board', owner: board }, name };
		return { key, value: '', credential: true, secret: refKey(ref) };
	});
}

// The draft of a stored one-off card, to edit it.
export function oneOffFrom(settings: PostItSettings, title: string, board: UUID): OneOffDraft {
	const bindings = settings.bindings ?? {};
	return {
		title,
		resource: settings.resource ?? '',
		response: settings.response === 'html' ? 'html' : 'json',
		rate: settings.rate,
		queries: rowsOf(settings.request?.queries, bindings, board),
		headers: rowsOf(settings.request?.headers, bindings, board),
		outputs: Object.entries(settings.query ?? {}).map(([key, query]) => ({ key, query }))
	};
}

export function outputKeys(draft: OneOffDraft) {
	return draft.outputs.map((o) => o.key.trim()).filter(Boolean);
}

// What the card is saved with. A credential row the user may no longer use
// (not in `usable`) is left out; problems() reports it first.
export function oneOffBody(
	draft: OneOffDraft,
	usable: SecretMeta[],
	board: UUID
): { title: { text: string; vars: boolean }; body: OneOffRequest } {
	const byKey = new Map(usable.map((s) => [refKey(s), s]));
	const picked: Record<string, SecretMeta> = {};
	const values = (rows: ValueRow[], prefix: string) => {
		const out: Record<string, string> = {};
		rows.forEach((row, i) => {
			const key = row.key.trim();
			if (!key) return;
			if (!row.credential) {
				out[key] = row.value.trim();
				return;
			}
			const secret = byKey.get(row.secret);
			if (!secret) return;
			// bindingsFor names each pick; the "$NAME" it answers is the value.
			const slot = `${prefix}${i}`;
			picked[slot] = secret;
			out[key] = slot;
		});
		return out;
	};

	const queries = values(draft.queries, 'q');
	const headers = values(draft.headers, 'h');
	const secrets = bindingsFor(picked, board);
	for (const record of [queries, headers]) {
		for (const [key, slot] of Object.entries(record)) {
			if (slot in secrets.params) record[key] = secrets.params[slot];
		}
	}

	const outputs = draft.outputs.map((o) => ({ key: o.key.trim(), query: o.query.trim() }));
	const text = draft.title.trim();
	return {
		title: {
			text,
			vars: analyzeTitle(
				text,
				outputs.map((o) => o.key)
			).vars
		},
		body: {
			resource: draft.resource.trim(),
			request: { method: 'GET', headers, queries },
			response: draft.response,
			query: Object.fromEntries(outputs.map((o) => [o.key, o.query])),
			rate: Math.max(0, Math.round(Number(draft.rate) || 0)),
			bindings: secrets.bindings
		}
	};
}

export type OneOffProblem =
	| { code: 'resource' | 'outputs' | 'output_query' | 'row_key' | 'credential' | 'title' }
	| { code: 'output_key' | 'duplicate'; value: string };

// What the backend would refuse, so the form can say it before saving.
export function oneOffProblems(draft: OneOffDraft, usable: SecretMeta[]): OneOffProblem[] {
	const found: OneOffProblem[] = [];
	const known = new Set(usable.map(refKey));

	try {
		const url = new URL(draft.resource.trim());
		if (url.protocol !== 'http:' && url.protocol !== 'https:') throw new Error();
	} catch {
		found.push({ code: 'resource' });
	}

	for (const rows of [draft.queries, draft.headers]) {
		const seen = new Set<string>();
		for (const row of rows) {
			const key = row.key.trim();
			// An untouched row is left out.
			if (!key && !row.value.trim() && !row.credential) continue;
			if (!key) found.push({ code: 'row_key' });
			else if (seen.has(key)) found.push({ code: 'duplicate', value: key });
			seen.add(key);
			if (row.credential && !known.has(row.secret)) found.push({ code: 'credential' });
		}
	}

	if (draft.outputs.length === 0) found.push({ code: 'outputs' });
	const outputs = new Set<string>();
	for (const o of draft.outputs) {
		const key = o.key.trim();
		if (!key) found.push({ code: 'output_key', value: key });
		else if (outputs.has(key)) found.push({ code: 'duplicate', value: key });
		outputs.add(key);
		if (!o.query.trim()) found.push({ code: 'output_query' });
	}

	const title = draft.title.trim();
	const analysis = analyzeTitle(title, [...outputs]);
	if (!title || analysis.unsupported || analysis.unknown.length > 0) found.push({ code: 'title' });

	// One message per problem is enough.
	return found.filter(
		(p, i) =>
			found.findIndex(
				(q) => q.code === p.code && ('value' in q ? q.value : '') === ('value' in p ? p.value : '')
			) === i
	);
}
