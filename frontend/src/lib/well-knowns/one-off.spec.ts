import { describe, expect, it } from 'vitest';
import { refKey } from '$lib/secrets/binding';
import type { PostItSettings, SecretMeta } from '$types/api';
import {
	bodyCredentialNames,
	emptyOneOff,
	insertCredential,
	MAX_REQUEST_BODY,
	nodeType,
	ONE_OFF,
	oneOffBody,
	oneOffFrom,
	oneOffProblems,
	type OneOffDraft
} from './one-off';

const board = 'b1';

function secret(kind: SecretMeta['scope']['kind'], owner: string, name: string): SecretMeta {
	return {
		scope: { kind, owner },
		name,
		kind: 'api_key',
		grants: [],
		authorized: true,
		created_at: ''
	} as unknown as SecretMeta;
}

const boardKey = secret('board', board, 'OWM_KEY');
const mine = secret('member', `${board}:u1`, 'MY_TOKEN');
const usable = [boardKey, mine];

function weather(): OneOffDraft {
	return {
		...emptyOneOff(),
		title: ' Now {{temp}}°C ',
		resource: ' https://api.openweathermap.org/data/2.5/weather ',
		queries: [
			{ key: 'q', value: ' Buenos Aires ', credential: false, secret: '' },
			{ key: 'appid', value: '', credential: true, secret: refKey(boardKey) },
			{ key: ' ', value: '', credential: false, secret: '' }
		],
		headers: [{ key: 'Authorization', value: '', credential: true, secret: refKey(mine) }],
		outputs: [{ key: ' temp ', query: ' .main.temp ' }]
	};
}

describe('nodeType', () => {
	it('names a card without a type', () => {
		expect(nodeType('')).toBe(ONE_OFF);
		expect(nodeType(undefined)).toBe(ONE_OFF);
		expect(nodeType('dog_facts')).toBe('dog_facts');
	});
});

describe('oneOffBody', () => {
	it('sends credentials as $NAME, binding only what the board scope cannot resolve', () => {
		const { title, body } = oneOffBody(weather(), usable, board);
		expect(title).toEqual({ text: 'Now {{temp}}°C', vars: true });
		expect(body).toEqual({
			resource: 'https://api.openweathermap.org/data/2.5/weather',
			request: {
				method: 'GET',
				queries: { q: 'Buenos Aires', appid: '$OWM_KEY' },
				headers: { Authorization: '$MY_TOKEN' }
			},
			response: 'json',
			query: { temp: '.main.temp' },
			rate: 600,
			bindings: { MY_TOKEN: { scope: mine.scope, name: 'MY_TOKEN' } }
		});
	});
});

describe('oneOffFrom', () => {
	it('round-trips a stored card', () => {
		const { title, body } = oneOffBody(weather(), usable, board);
		const settings: PostItSettings = {
			id: 'p1',
			board,
			params: null,
			rate: body.rate,
			bindings: body.bindings,
			resource: body.resource,
			request: body.request,
			response: body.response,
			query: body.query
		};
		const draft = oneOffFrom(settings, title.text, board);
		expect(oneOffBody(draft, usable, board).body).toEqual(body);
		expect(draft.queries.find((r) => r.key === 'appid')).toMatchObject({
			credential: true,
			secret: refKey(boardKey)
		});
	});
});

describe('oneOffProblems', () => {
	it('accepts a valid card', () => {
		expect(oneOffProblems(weather(), usable)).toEqual([]);
	});

	it('reports what the backend would refuse', () => {
		const draft: OneOffDraft = {
			...emptyOneOff(),
			resource: 'ftp://example.com',
			queries: [
				{ key: 'a', value: '1', credential: false, secret: '' },
				{ key: 'a', value: '2', credential: false, secret: '' },
				{ key: 'k', value: '', credential: true, secret: 'gone' }
			],
			headers: [{ key: '', value: 'x', credential: false, secret: '' }],
			outputs: [{ key: 'x', query: '' }],
			title: '{{nope}}'
		};
		expect(oneOffProblems(draft, usable).map((p) => p.code)).toEqual([
			'resource',
			'duplicate',
			'credential',
			'row_key',
			'output_query',
			'title'
		]);
	});
});

describe('POST cards', () => {
	function search(): OneOffDraft {
		return {
			...emptyOneOff(),
			title: 'Desde {{price}} USD',
			resource: 'https://ignav.com/api/fares/one-way',
			method: 'POST',
			body: ' {"origin":"EZE","destination":"NRT","key":"$OWM_KEY","partner":"$MY_TOKEN"} ',
			bodySecrets: { MY_TOKEN: refKey(mine) },
			outputs: [{ key: 'price', query: '.itineraries[0].price.amount' }]
		};
	}

	it('sends the body with the POST and binds what the board scope cannot resolve', () => {
		const { body } = oneOffBody(search(), usable, board);
		expect(body.request).toEqual({
			method: 'POST',
			headers: {},
			queries: {},
			body: '{"origin":"EZE","destination":"NRT","key":"$OWM_KEY","partner":"$MY_TOKEN"}'
		});
		expect(body.bindings).toEqual({ MY_TOKEN: { scope: mine.scope, name: 'MY_TOKEN' } });
	});

	it('drops the body of a GET', () => {
		const { body } = oneOffBody({ ...search(), method: 'GET' }, usable, board);
		expect(body.request).toEqual({ method: 'GET', headers: {}, queries: {} });
	});

	it('round-trips a stored POST card', () => {
		const { body } = oneOffBody(search(), usable, board);
		const settings: PostItSettings = {
			id: 'p1',
			board,
			params: null,
			rate: body.rate,
			bindings: body.bindings,
			resource: body.resource,
			request: { ...body.request, headers: {}, queries: {} },
			response: body.response,
			query: body.query
		};
		const draft = oneOffFrom(settings, 'Desde {{price}} USD', board);
		expect(draft.method).toBe('POST');
		expect(draft.body).toBe(body.request.body);
		expect(draft.bodySecrets).toEqual({ OWM_KEY: refKey(boardKey), MY_TOKEN: refKey(mine) });
	});

	it('finds the credentials the body names, only as whole values', () => {
		expect(bodyCredentialNames('{"a":"$ONE","b":["$TWO",{"c":"$ONE"}],"d":"not $THREE","e":"$lower"}')).toEqual([
			'ONE',
			'TWO'
		]);
		expect(bodyCredentialNames('origin=EZE')).toBeNull();
	});

	it('inserts a credential at the cursor', () => {
		expect(insertCredential('{"key":}', 7, 'IGNAV')).toEqual({ body: '{"key":"$IGNAV"}', cursor: 15 });
		expect(insertCredential('', 99, 'X').body).toBe('"$X"');
	});

	it('reports a body the backend would refuse', () => {
		const codes = (draft: OneOffDraft) => oneOffProblems(draft, usable).map((p) => p.code);
		expect(codes(search())).toEqual([]);
		expect(codes({ ...search(), body: 'origin=EZE' })).toContain('body_json');
		expect(codes({ ...search(), body: JSON.stringify('a'.repeat(MAX_REQUEST_BODY)) })).toContain('body_size');
		expect(oneOffProblems({ ...search(), body: '{"k":"$NOBODY"}' }, usable)).toContainEqual({
			code: 'body_credential',
			value: 'NOBODY'
		});
		// A GET ignores whatever was typed in the body.
		expect(codes({ ...search(), method: 'GET', body: 'origin=EZE' })).toEqual([]);
	});
});
