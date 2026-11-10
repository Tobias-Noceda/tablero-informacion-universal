import { describe, expect, it } from 'vitest';
import type { PostitTemplate } from '$types/api';
import { bodyOf, draftFrom, emptyDraft, problems, type Draft } from './draft';

function catFact(): Draft {
	return {
		...emptyDraft(),
		name: ' Cat fact ',
		resource: 'https://catfact.ninja/fact',
		queries: [
			{ key: 'max_length', value: '$max' },
			{ key: ' ', value: 'ignored' }
		],
		parameters: [{ key: '$max', label: 'Max', type: 'number', placeholder: '', default: '140' }],
		outputs: [{ key: 'fact', label: 'Fact', query: '.fact', example: 'Cats sleep' }],
		title: '{{fact}}'
	};
}

describe('bodyOf', () => {
	it('builds what the backend expects', () => {
		expect(bodyOf(catFact())).toEqual({
			name: 'Cat fact',
			description: undefined,
			title: { text: '{{fact}}', vars: true },
			resource: 'https://catfact.ninja/fact',
			request: { method: 'GET', headers: {}, queries: { max_length: '$max' } },
			response: 'json',
			query: { fact: '.fact' },
			rate: 600,
			parameters: [
				{ key: '$max', label: 'Max', type: 'number', placeholder: undefined, default: '140' }
			],
			outputs: [{ key: 'fact', label: 'Fact', example: 'Cats sleep' }]
		});
	});

	it('round-trips through a stored definition', () => {
		const body = bodyOf(catFact());
		const stored: PostitTemplate = {
			...body,
			key: 'custom_x',
			scope: { kind: 'user', owner: 'ana' },
			created_by: 'ana',
			created_at: '',
			updated_by: 'ana',
			updated_at: ''
		};
		expect(bodyOf(draftFrom(stored))).toEqual(body);
	});
});

describe('problems', () => {
	it('accepts a valid draft', () => {
		expect(problems(catFact())).toEqual([]);
	});

	it('mirrors the backend rules', () => {
		const draft = catFact();
		draft.name = '';
		draft.resource = 'ftp://example.com';
		draft.parameters.push({ key: '$MAX', label: '', type: 'string', placeholder: '', default: '' });
		draft.headers = [{ key: 'X-Key', value: '$token' }];
		draft.outputs.push({ key: 'fact', label: 'Again', query: '', example: '' });
		draft.title = '{{nope}}';

		const codes = problems(draft).map((p) => ('value' in p ? `${p.code}:${p.value}` : p.code));
		expect(codes).toEqual([
			'name',
			'resource',
			'param_key:$MAX',
			'param_label',
			'duplicate:fact',
			'output_query',
			'undeclared:$token',
			'title'
		]);
	});

	it('needs at least one output', () => {
		const draft = catFact();
		draft.outputs = [];
		draft.title = 'Static';
		expect(problems(draft)).toEqual([{ code: 'outputs' }]);
	});
});
