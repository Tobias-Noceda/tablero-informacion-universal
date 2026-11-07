import { analyzeTitle } from '$lib/title';
import type { PostitTemplate, PostitTemplateBody, PostitTemplateParameter } from '$types/api';

// The post-it template form's state, and what it sends. Kept apart from the
// component so the rules the backend enforces can be mirrored and tested.

export type Row = { key: string; value: string };
export type ParameterRow = {
	key: string;
	label: string;
	type: PostitTemplateParameter['type'];
	placeholder: string;
	default: string;
};
export type OutputRow = { key: string; label: string; query: string; example: string };

export type Draft = {
	name: string;
	description: string;
	resource: string;
	response: 'json' | 'html';
	rate: number;
	headers: Row[];
	queries: Row[];
	parameters: ParameterRow[];
	outputs: OutputRow[];
	title: string;
};

export const PLACEHOLDER = /^\$[a-z][a-z0-9_]*$/;

export function emptyDraft(): Draft {
	return {
		name: '',
		description: '',
		resource: '',
		response: 'json',
		rate: 600,
		headers: [],
		queries: [],
		parameters: [],
		outputs: [{ key: '', label: '', query: '', example: '' }],
		title: ''
	};
}

const rows = (record: Record<string, string> | null | undefined): Row[] =>
	Object.entries(record ?? {}).map(([key, value]) => ({ key, value }));

const record = (list: Row[]) =>
	Object.fromEntries(list.filter((r) => r.key.trim()).map((r) => [r.key.trim(), r.value.trim()]));

export function draftFrom(template: PostitTemplate): Draft {
	return {
		name: template.name,
		description: template.description ?? '',
		resource: template.resource,
		response: template.response,
		rate: template.rate,
		headers: rows(template.request.headers),
		queries: rows(template.request.queries),
		parameters: (template.parameters ?? []).map((p) => ({
			key: p.key,
			label: p.label,
			type: p.type,
			placeholder: p.placeholder ?? '',
			default: p.default ?? ''
		})),
		outputs: (template.outputs ?? []).map((o) => ({
			key: o.key,
			label: o.label,
			query: template.query?.[o.key] ?? '',
			example: o.example === undefined || o.example === null ? '' : String(o.example)
		})),
		title: template.title.text
	};
}

export function bodyOf(draft: Draft): PostitTemplateBody {
	const outputs = draft.outputs.map((o) => ({ ...o, key: o.key.trim() }));
	const title = draft.title.trim();
	return {
		name: draft.name.trim(),
		description: draft.description.trim() || undefined,
		title: {
			text: title,
			vars: analyzeTitle(
				title,
				outputs.map((o) => o.key)
			).vars
		},
		resource: draft.resource.trim(),
		request: { method: 'GET', headers: record(draft.headers), queries: record(draft.queries) },
		response: draft.response,
		query: Object.fromEntries(outputs.map((o) => [o.key, o.query.trim()])),
		rate: Math.max(0, Math.round(Number(draft.rate) || 0)),
		parameters: draft.parameters.map((p) => ({
			key: p.key.trim(),
			label: p.label.trim(),
			type: p.type,
			placeholder: p.placeholder.trim() || undefined,
			default: p.default.trim() || undefined
		})),
		outputs: outputs.map((o) => ({
			key: o.key,
			label: o.label.trim(),
			example: o.example.trim() || undefined
		}))
	};
}

export type Problem =
	| {
			code:
				| 'name'
				| 'resource'
				| 'outputs'
				| 'output_query'
				| 'param_label'
				| 'output_label'
				| 'title';
	  }
	| { code: 'param_key' | 'output_key' | 'duplicate' | 'undeclared'; value: string };

// What the backend would refuse, so the form can say it before saving.
export function problems(draft: Draft): Problem[] {
	const found: Problem[] = [];
	const body = bodyOf(draft);

	if (!body.name) found.push({ code: 'name' });

	try {
		const url = new URL(body.resource);
		if (url.protocol !== 'http:' && url.protocol !== 'https:') throw new Error();
	} catch {
		found.push({ code: 'resource' });
	}

	const params = new Set<string>();
	for (const p of body.parameters) {
		if (!PLACEHOLDER.test(p.key)) found.push({ code: 'param_key', value: p.key });
		else if (params.has(p.key)) found.push({ code: 'duplicate', value: p.key });
		params.add(p.key);
		if (!p.label) found.push({ code: 'param_label' });
	}

	if (body.outputs.length === 0) found.push({ code: 'outputs' });
	const outputs = new Set<string>();
	body.outputs.forEach((o, i) => {
		if (!o.key) found.push({ code: 'output_key', value: o.key });
		else if (outputs.has(o.key)) found.push({ code: 'duplicate', value: o.key });
		outputs.add(o.key);
		if (!o.label) found.push({ code: 'output_label' });
		if (!draft.outputs[i].query.trim()) found.push({ code: 'output_query' });
	});

	for (const value of [
		...Object.values(body.request.headers),
		...Object.values(body.request.queries)
	]) {
		if (value.startsWith('$') && !params.has(value)) found.push({ code: 'undeclared', value });
	}

	const title = analyzeTitle(body.title.text, [...outputs]);
	if (!body.title.text || title.unsupported || title.unknown.length > 0)
		found.push({ code: 'title' });

	// One message per problem is enough.
	return found.filter(
		(p, i) =>
			found.findIndex(
				(q) => q.code === p.code && ('value' in q ? q.value : '') === ('value' in p ? p.value : '')
			) === i
	);
}
