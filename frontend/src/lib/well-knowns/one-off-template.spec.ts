import { describe, expect, it } from 'vitest';

import { emptyOneOff, type OneOffDraft } from './one-off';
import { canBecomeTemplate, nameFrom, templateDraftFrom } from './one-off-template';

function card(edit: Partial<OneOffDraft> = {}): OneOffDraft {
	return {
		...emptyOneOff(),
		title: 'Bitcoin: {{price}} USD',
		resource: ' https://api.example.com/price ',
		rate: 900,
		queries: [{ key: 'ids', value: 'bitcoin', credential: false, secret: '' }],
		headers: [
			{ key: 'Accept', value: 'application/json', credential: false, secret: '' },
			{ key: 'Authorization', value: '', credential: true, secret: 'board:b:API_KEY' }
		],
		outputs: [{ key: 'price', query: '.bitcoin.usd' }],
		...edit
	};
}

describe('templateDraftFrom', () => {
	it('fills the card type in from the card', () => {
		const draft = templateDraftFrom(card());
		expect(draft).toMatchObject({
			name: 'Bitcoin: USD',
			resource: 'https://api.example.com/price',
			response: 'json',
			rate: 900,
			title: 'Bitcoin: {{price}} USD',
			queries: [{ key: 'ids', value: 'bitcoin' }],
			outputs: [{ key: 'price', label: 'price', query: '.bitcoin.usd', example: '' }]
		});
	});

	it('turns each credential into a parameter whoever uses the type picks', () => {
		const draft = templateDraftFrom(
			card({
				queries: [{ key: 'token', value: '', credential: true, secret: 'mine:X' }]
			})
		);
		expect(draft.headers).toEqual([
			{ key: 'Accept', value: 'application/json' },
			{ key: 'Authorization', value: '$credential' }
		]);
		expect(draft.queries).toEqual([{ key: 'token', value: '$credential_2' }]);
		expect(draft.parameters).toEqual([
			{ key: '$credential', label: 'Authorization', type: 'secret', placeholder: '', default: '' },
			{ key: '$credential_2', label: 'token', type: 'secret', placeholder: '', default: '' }
		]);
	});

	it('leaves out empty rows, but keeps one output row to fill in', () => {
		const draft = templateDraftFrom(
			card({
				headers: [{ key: ' ', value: 'x', credential: false, secret: '' }],
				outputs: [{ key: '', query: '' }]
			})
		);
		expect(draft.headers).toEqual([]);
		expect(draft.outputs).toEqual([{ key: '', label: '', query: '', example: '' }]);
	});
});

describe('nameFrom', () => {
	it('drops the values of the title', () => {
		expect(nameFrom('{{city}} weather: {{temp}}°C')).toBe('weather: °C');
		expect(nameFrom('Price {{p}}')).toBe('Price');
		expect(nameFrom('{{only}}')).toBe('');
	});
});

describe('canBecomeTemplate', () => {
	it('takes GET requests only', () => {
		expect(canBecomeTemplate({ method: 'GET' })).toBe(true);
		expect(canBecomeTemplate({ method: 'POST' })).toBe(false);
	});
});
