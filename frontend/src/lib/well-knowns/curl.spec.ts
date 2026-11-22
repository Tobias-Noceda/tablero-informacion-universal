import { describe, expect, it } from 'vitest';

import { CurlError, fillFromCurl, parseCurl, tokenize } from './curl';
import { emptyOneOff } from './one-off';

describe('tokenize', () => {
	it('splits like a shell', () => {
		expect(tokenize(`curl -H 'A: b c' "x\\"y" plain\\ word`)).toEqual([
			'curl',
			'-H',
			'A: b c',
			'x"y',
			'plain word'
		]);
	});

	it('joins continued lines, from bash and from cmd.exe', () => {
		expect(tokenize('curl \\\n  -s \\\r\n  https://a.b')).toEqual(['curl', '-s', 'https://a.b']);
		expect(tokenize('curl ^\n  https://a.b')).toEqual(['curl', 'https://a.b']);
	});

	it('refuses an unclosed quote', () => {
		expect(() => tokenize(`curl 'https://a.b`)).toThrow(CurlError);
	});
});

describe('parseCurl', () => {
	it('reads a GET with its query string and headers', () => {
		const got = parseCurl(
			`curl -s "https://api.example.com/v1/price?ids=bitcoin&vs=usd" -H "Accept: application/json"`
		);
		expect(got).toEqual({
			resource: 'https://api.example.com/v1/price',
			method: 'GET',
			queries: [
				{ key: 'ids', value: 'bitcoin', credential: false },
				{ key: 'vs', value: 'usd', credential: false }
			],
			headers: [{ key: 'Accept', value: 'application/json', credential: false }],
			body: '',
			credentials: [],
			dropped: []
		});
	});

	it('never keeps a credential as text', () => {
		const got = parseCurl(
			`curl 'https://api.example.com/data?api_key=sk_live_123&q=x' -H 'Authorization: Bearer abc' -H 'X-Api-Key: zzz'`
		);
		expect(got.queries).toEqual([
			{ key: 'api_key', value: '', credential: true },
			{ key: 'q', value: 'x', credential: false }
		]);
		expect(got.headers.every((h) => h.credential && h.value === '')).toBe(true);
		expect(got.credentials).toEqual(['Authorization', 'X-Api-Key', 'api_key']);
		expect(JSON.stringify(got)).not.toMatch(/sk_live_123|abc|zzz/);
	});

	it('turns -u into a credential to pick', () => {
		const got = parseCurl('curl -u user:pass https://a.example.com');
		expect(got.headers).toEqual([{ key: 'Authorization', value: '', credential: true }]);
		expect(JSON.stringify(got)).not.toContain('pass');
	});

	it('reads a POST with a JSON body', () => {
		const got = parseCurl(
			`curl -X POST https://api.example.com/search -H 'Content-Type: application/json' -d '{"q": "cats"}'`
		);
		expect(got.method).toBe('POST');
		expect(got.body).toBe('{"q": "cats"}');
	});

	it('takes a body without -X as a POST, as curl does', () => {
		expect(parseCurl(`curl https://a.example.com --data-raw '{"a":1}'`).method).toBe('POST');
	});

	it('adds the JSON headers --json implies', () => {
		const got = parseCurl(`curl --json '{"a":1}' https://a.example.com`);
		expect(got.method).toBe('POST');
		expect(got.headers).toEqual([
			{ key: 'Content-Type', value: 'application/json', credential: false },
			{ key: 'Accept', value: 'application/json', credential: false }
		]);
	});

	it('moves -G data to the query string', () => {
		const got = parseCurl('curl -G https://a.example.com/s -d q=cats -d limit=5');
		expect(got.method).toBe('GET');
		expect(got.queries.map((q) => [q.key, q.value])).toEqual([
			['q', 'cats'],
			['limit', '5']
		]);
		expect(got.body).toBe('');
	});

	it('says what a card cannot send', () => {
		expect(parseCurl('curl -X DELETE https://a.example.com/1').dropped).toContain('method');
		expect(parseCurl('curl -d a=1&b=2 https://a.example.com').dropped).toContain('body_not_json');
		expect(parseCurl('curl -o out.json --retry 3 https://a.example.com').dropped).toContain(
			'flags'
		);
	});

	it('reads --flag=value and --url', () => {
		const got = parseCurl(
			'curl --request=POST --url=https://a.example.com/x --header=Accept:text/plain'
		);
		expect(got.method).toBe('POST');
		expect(got.resource).toBe('https://a.example.com/x');
		expect(got.headers).toEqual([{ key: 'Accept', value: 'text/plain', credential: false }]);
	});

	it('adds https to a URL without a scheme', () => {
		expect(parseCurl('curl api.example.com/v1').resource).toBe('https://api.example.com/v1');
	});

	it('refuses what is not a cURL command with a URL', () => {
		expect(() => parseCurl('wget https://a.example.com')).toThrow(CurlError);
		expect(() => parseCurl('curl -s')).toThrow(CurlError);
	});
});

describe('fillFromCurl', () => {
	it('puts the request in the form and keeps the rest', () => {
		const draft = {
			...emptyOneOff(),
			title: 'Price {{price}}',
			outputs: [{ key: 'price', query: '.price' }],
			headers: [{ key: 'Old', value: 'x', credential: false, secret: '' }]
		};
		const filled = fillFromCurl(
			draft,
			parseCurl("curl 'https://a.example.com/p?ids=1' -H 'Authorization: Bearer t'")
		);
		expect(filled).toMatchObject({
			title: 'Price {{price}}',
			outputs: [{ key: 'price', query: '.price' }],
			resource: 'https://a.example.com/p',
			method: 'GET',
			queries: [{ key: 'ids', value: '1', credential: false, secret: '' }],
			headers: [{ key: 'Authorization', value: '', credential: true, secret: '' }]
		});
	});
});
