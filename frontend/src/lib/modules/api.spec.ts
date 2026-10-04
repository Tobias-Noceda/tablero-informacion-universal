import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import * as api from './api.svelte';

type Call = { url: string; init: RequestInit };

function stubFetch(...statuses: number[]) {
	const calls: Call[] = [];
	const fetchMock = vi.fn(async (url: URL | string, init: RequestInit = {}) => {
		calls.push({ url: String(url), init });
		const status = statuses[Math.min(calls.length, statuses.length) - 1];
		return new Response(JSON.stringify({ status }), { status });
	});
	vi.stubGlobal('fetch', fetchMock);
	return calls;
}

function authorization(call: Call) {
	return new Headers(call.init.headers).get('Authorization');
}

class FakeAuthenticator implements api.Authenticator {
	token: string | null = 'first';
	renewals = 0;
	expirations = 0;

	constructor(private readonly renewed: string | null = 'second') {}

	header() {
		return this.token && `Bearer ${this.token}`;
	}

	async renew() {
		this.renewals++;
		this.token = this.renewed;
		return this.renewed !== null;
	}

	expired() {
		this.expirations++;
	}
}

describe('api client', () => {
	let auth: FakeAuthenticator;

	beforeEach(() => {
		vi.stubGlobal('location', new URL('https://tiu.example/board/1'));
		auth = new FakeAuthenticator();
		api.authenticate(auth);
	});

	afterEach(() => {
		api.authenticate(null);
		vi.unstubAllGlobals();
	});

	it('sends relative paths to /api on the page origin with the bearer', async () => {
		const calls = stubFetch(200);

		await api.post('/v1/boards', { name: 'x' });

		expect(calls[0].url).toBe('https://tiu.example/api/v1/boards');
		expect(calls[0].init.method).toBe('POST');
		expect(calls[0].init.body).toBe('{"name":"x"}');
		expect(authorization(calls[0])).toBe('Bearer first');
		expect(new Headers(calls[0].init.headers).get('Content-Type')).toBe('application/json');
	});

	it('sends no Authorization header without a session', async () => {
		const calls = stubFetch(200);
		auth.token = null;

		await api.get('/v1/boards');

		expect(authorization(calls[0])).toBeNull();
	});

	it('renews once on 401 and retries with the new token', async () => {
		const calls = stubFetch(401, 200);

		const response = await api.patch('/v1/users/1', { name: 'Ana' });

		expect(response.status).toBe(200);
		expect(auth.renewals).toBe(1);
		expect(calls.map(authorization)).toEqual(['Bearer first', 'Bearer second']);
		expect(calls[1].init.body).toBe('{"name":"Ana"}');
		expect(auth.expirations).toBe(0);
	});

	it('gives up after one retry and reports the session as expired', async () => {
		const calls = stubFetch(401, 401, 401);

		const response = await api.del('/v1/boards/1');

		expect(response.status).toBe(401);
		expect(calls).toHaveLength(2);
		expect(auth.renewals).toBe(1);
		expect(auth.expirations).toBe(1);
	});

	it('reports the session as expired when it cannot be renewed', async () => {
		auth = new FakeAuthenticator(null);
		api.authenticate(auth);
		const calls = stubFetch(401);

		await expect(api.get('/v1/users/1')).rejects.toMatchObject({ status: 401 });

		expect(calls).toHaveLength(1);
		expect(auth.expirations).toBe(1);
	});

	it('never retries the auth endpoints themselves', async () => {
		const calls = stubFetch(401);

		const response = await api.post('/v1/auth/login', { email: 'a', password: 'b' });

		expect(response.status).toBe(401);
		expect(calls).toHaveLength(1);
		expect(auth.renewals).toBe(0);
		expect(auth.expirations).toBe(0);
	});

	it('turns network failures on get into a 503', async () => {
		vi.stubGlobal('fetch', vi.fn().mockRejectedValue(new TypeError('offline')));
		vi.spyOn(console, 'error').mockImplementation(() => {});

		await expect(api.get('/v1/boards')).rejects.toMatchObject({ status: 503 });
	});
});
