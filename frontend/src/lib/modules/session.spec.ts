import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import type { Profile, SessionResponse } from '$types/api';
import { AuthError } from '$services/auth';
import * as api from './api.svelte';

const goto = vi.hoisted(() => vi.fn());
vi.mock('$app/navigation', () => ({ goto }));

const { Session } = await import('./session.svelte');

const ana: Profile = {
	id: '0b9a4c5e-1111-4222-8333-944445555666',
	email: 'ana@example.com',
	email_verified: true,
	name: 'Ana',
	admin: false,
	identities: ['password'],
	created_at: '2026-10-04T00:00:00Z'
};

function sessionBody(token: string): SessionResponse {
	return { access_token: token, token_type: 'Bearer', expires_in: 900, user: ana };
}

type Route = { status: number; body?: unknown };

// The backend, as a table of "METHOD /path" → answer (or a queue of answers).
function backend(routes: Record<string, Route | Route[] | (() => Promise<Route>)>) {
	const calls: string[] = [];
	vi.stubGlobal(
		'fetch',
		vi.fn(async (url: URL, init: RequestInit = {}) => {
			const key = `${init.method} ${new URL(url).pathname.replace(/^\/api/, '')}`;
			calls.push(key);
			let route = routes[key];
			if (typeof route === 'function') route = await route();
			if (Array.isArray(route)) route = route.length > 1 ? route.shift()! : route[0];
			if (!route) throw new Error(`unexpected ${key}`);
			return new Response(route.body === undefined ? null : JSON.stringify(route.body), {
				status: route.status
			});
		})
	);
	return calls;
}

let channels = 0;
const sessions: InstanceType<typeof Session>[] = [];

function newSession(channel = `tiu-session-test-${++channels}`) {
	const session = new Session(channel);
	sessions.push(session);
	return session;
}

describe('session', () => {
	beforeEach(() => {
		vi.stubGlobal('location', new URL('https://tiu.example/board/7?x=1'));
		goto.mockReset();
	});

	afterEach(() => {
		sessions.splice(0).forEach((s) => s.dispose());
		api.authenticate(null);
		vi.unstubAllGlobals();
	});

	it('starts unknown and bootstraps from the refresh cookie', async () => {
		const calls = backend({ 'POST /v1/auth/refresh': { status: 200, body: sessionBody('t1') } });
		const session = newSession();
		expect(session.status).toBe('unknown');

		await Promise.all([session.bootstrap(), session.bootstrap()]);
		await session.bootstrap();

		expect(calls).toEqual(['POST /v1/auth/refresh']);
		expect(session.status).toBe('authenticated');
		expect(session.user?.id).toBe(ana.id);
		expect(session.header()).toBe('Bearer t1');
		expect(session.expiresAt).toBeGreaterThan(Date.now() + 890_000);
	});

	it('is anonymous when there is no cookie', async () => {
		backend({ 'POST /v1/auth/refresh': { status: 401, body: { error: 'unauthorized' } } });
		const session = newSession();

		await session.bootstrap();

		expect(session.status).toBe('anonymous');
		expect(session.user).toBeNull();
		expect(session.header()).toBeNull();
	});

	it('shares one refresh between concurrent callers', async () => {
		let release!: () => void;
		const gate = new Promise<void>((r) => (release = r));
		const calls = backend({
			'POST /v1/auth/refresh': async () => {
				await gate;
				return { status: 200, body: sessionBody('t2') };
			}
		});
		const session = newSession();

		const pending = [session.refresh(), session.refresh(), session.renew()];
		release();

		expect(await Promise.all(pending)).toEqual([true, true, true]);
		expect(calls).toHaveLength(1);
		expect(session.header()).toBe('Bearer t2');
	});

	it('forgets the session when a refresh fails', async () => {
		backend({
			'POST /v1/auth/login': { status: 200, body: sessionBody('t1') },
			'POST /v1/auth/refresh': { status: 401, body: { error: 'unauthorized' } }
		});
		const session = newSession();
		await session.login('ana@example.com', 'pw');

		expect(await session.refresh()).toBe(false);

		expect(session.status).toBe('anonymous');
		expect(session.header()).toBeNull();
	});

	it('signs in and reports backend refusals by code', async () => {
		backend({
			'POST /v1/auth/login': [
				{ status: 401, body: { error: 'invalid_credentials' } },
				{ status: 200, body: sessionBody('t1') }
			]
		});
		const session = newSession();

		const refused = await session.login('ana@example.com', 'wrong').catch((e) => e);
		expect(refused).toBeInstanceOf(AuthError);
		expect(refused).toMatchObject({ code: 'invalid_credentials', status: 401 });
		expect(session.status).not.toBe('authenticated');

		await session.login('ana@example.com', 'right');
		expect(session.status).toBe('authenticated');
		expect(session.header()).toBe('Bearer t1');
	});

	it('finishes a google sign-in and hands back where to go', async () => {
		backend({
			'POST /v1/auth/google/callback': {
				status: 200,
				body: { ...sessionBody('g1'), next: '/board/9' }
			}
		});
		const session = newSession();

		expect(await session.googleCallback('code', 'state')).toBe('/board/9');
		expect(session.header()).toBe('Bearer g1');
	});

	it('logs out even if the backend is unreachable', async () => {
		const calls = backend({ 'POST /v1/auth/login': { status: 200, body: sessionBody('t1') } });
		const session = newSession();
		await session.login('ana@example.com', 'pw');

		await session.logout();

		expect(calls).toContain('POST /v1/auth/logout');
		expect(session.status).toBe('anonymous');
		expect(session.user).toBeNull();
	});

	it('sends the user to sign in, back to where they were, when the session expires', () => {
		const session = newSession();

		session.expired();

		expect(session.status).toBe('anonymous');
		expect(goto).toHaveBeenCalledWith('/login?next=%2Fboard%2F7%3Fx%3D1');
	});

	it('lends its bearer to the api client', async () => {
		const calls: (string | null)[] = [];
		backend({ 'POST /v1/auth/login': { status: 200, body: sessionBody('t1') } });
		const session = newSession();
		await session.login('ana@example.com', 'pw');
		api.authenticate(session);
		vi.stubGlobal(
			'fetch',
			vi.fn(async (_url: URL, init: RequestInit) => {
				calls.push(new Headers(init.headers).get('Authorization'));
				return new Response('{}', { status: 200 });
			})
		);

		await api.get('/v1/boards');

		expect(calls).toEqual(['Bearer t1']);
	});

	it('follows other tabs: a sign-in and a sign-out elsewhere apply here', async () => {
		backend({
			'POST /v1/auth/refresh': { status: 401 },
			'POST /v1/auth/login': { status: 200, body: sessionBody('shared') }
		});
		const here = newSession('tiu-session-tabs');
		const there = newSession('tiu-session-tabs');
		await here.bootstrap();
		await there.bootstrap();

		await there.login('ana@example.com', 'pw');
		await vi.waitFor(() => expect(here.status).toBe('authenticated'));
		expect(here.header()).toBe('Bearer shared');

		await there.logout();
		await vi.waitFor(() => expect(here.status).toBe('anonymous'));
		expect(here.header()).toBeNull();
	});
});
