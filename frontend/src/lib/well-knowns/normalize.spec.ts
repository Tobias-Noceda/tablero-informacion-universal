import { describe, expect, it } from 'vitest';
import type { BoardWellKnown } from '$types/api';
import { CUSTOM_ORDER, toWellKnown } from './normalize';

const custom: BoardWellKnown = {
	key: 'custom_abc',
	scope: { kind: 'user', owner: 'ana' },
	name: 'Dato de gato',
	description: '',
	title: { text: '{{fact}}', vars: true },
	resource: 'https://catfact.ninja/fact',
	request: { method: 'GET', headers: null, queries: { max_length: '$max' } },
	response: 'json',
	query: { fact: '.fact' },
	rate: 60,
	parameters: [{ key: '$max', label: 'Largo', type: 'number', placeholder: '140' }],
	outputs: [{ key: 'fact', label: 'Dato', example: 'Duermen' }],
	created_by: 'ana',
	created_at: '2026-11-03T00:00:00Z',
	updated_by: 'ana',
	updated_at: '2026-11-03T00:00:00Z',
	usable: false
};

describe('toWellKnown', () => {
	it('copies its one language into both', () => {
		const wk = toWellKnown(custom);
		expect(wk.name_en).toBe('Dato de gato');
		expect(wk.name_es).toBe('Dato de gato');
		expect(wk.title_en).toBe('{{fact}}');
		expect(wk.title_es).toBe('{{fact}}');
		expect(wk.description_en).toBeUndefined();
		expect(wk.parameters).toEqual([
			{
				key: '$max',
				label_en: 'Largo',
				label_es: 'Largo',
				type: 'number',
				placeholder_en: '140',
				placeholder_es: '140',
				default: undefined
			}
		]);
		expect(wk.outputs).toEqual([
			{
				key: 'fact',
				label_en: 'Dato',
				label_es: 'Dato',
				example_en: 'Duermen',
				example_es: 'Duermen'
			}
		]);
	});

	it('keeps where it comes from and sorts after the catalogue', () => {
		const wk = toWellKnown(custom);
		expect(wk.scope).toEqual({ kind: 'user', owner: 'ana' });
		expect(wk.usable).toBe(false);
		expect(wk.builtin).toBe(false);
		expect(wk.order).toBe(CUSTOM_ORDER);
	});

	it('treats a type without a board as usable and tolerates null lists', () => {
		const plain: Omit<BoardWellKnown, 'usable'> & { usable?: boolean } = { ...custom };
		delete plain.usable;
		const wk = toWellKnown({ ...plain, parameters: null, outputs: null });
		expect(wk.usable).toBe(true);
		expect(wk.parameters).toEqual([]);
		expect(wk.outputs).toEqual([]);
	});
});
