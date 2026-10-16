import * as api from '$modules/api.svelte';
import type { GoogleSessionResponse, SessionResponse } from '$types/api';

// What the backend said went wrong, as the stable code in its `error` field
// (invalid_credentials, email_not_verified, rate_limited, invalid_token...).
export class AuthError extends Error {
	constructor(
		readonly code: string,
		readonly status: number
	) {
		super(code);
		this.name = 'AuthError';
	}
}

async function expect<T>(response: Response, status: number): Promise<T> {
	const body = await response.json().catch(() => ({}));
	if (response.status !== status) {
		throw new AuthError(typeof body?.error === 'string' ? body.error : 'unknown', response.status);
	}
	return body as T;
}

export async function register(email: string, password: string, name: string) {
	await expect(await api.post('/v1/auth/register', { email, password, name }), 202);
}

export async function verifyEmail(token: string) {
	return expect<SessionResponse>(await api.post('/v1/auth/verify-email', { token }), 200);
}

export async function login(email: string, password: string) {
	return expect<SessionResponse>(await api.post('/v1/auth/login', { email, password }), 200);
}

export async function refresh() {
	return expect<SessionResponse>(await api.post('/v1/auth/refresh', undefined), 200);
}

export async function logout() {
	await api.post('/v1/auth/logout', undefined);
}

export async function forgotPassword(email: string) {
	await expect(await api.post('/v1/auth/password/forgot', { email }), 202);
}

export async function resetPassword(token: string, password: string) {
	return expect<SessionResponse>(
		await api.post('/v1/auth/password/reset', { token, password }),
		200
	);
}

export async function googleStart(next: string) {
	const response = await api.request(
		'GET',
		`/v1/auth/google/start?${new URLSearchParams({ next })}`
	);
	const { authorization_url } = await expect<{ authorization_url: string }>(response, 200);
	return authorization_url;
}

export async function googleCallback(code: string, state: string) {
	return expect<GoogleSessionResponse>(
		await api.post('/v1/auth/google/callback', { code, state }),
		200
	);
}
