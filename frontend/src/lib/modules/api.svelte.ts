import { error } from '@sveltejs/kit';

type Path = `/${string}`;

// The session plugs itself in here (it calls this module, so it cannot be
// imported back): it lends the bearer and renews it when the API says 401.
export interface Authenticator {
	header(): string | null;
	renew(): Promise<boolean>;
	expired(): void;
}

let authenticator: Authenticator | null = null;

export function authenticate(next: Authenticator | null) {
	authenticator = next;
}

// The session protocol itself answers 401 for a bad password or a dead
// cookie; renewing and retrying those would only hide the answer.
const SESSION_PROTOCOL = '/v1/auth/';

function resolvePath(path: Path) {
	const base = import.meta.env.VITE_API_URL || globalThis.location?.href || 'http://localhost';
	return new URL('/api' + path, base);
}

export async function request(
	method: string,
	path: Path,
	body?: unknown,
	options?: RequestInit,
	fetchFn: typeof fetch = fetch
): Promise<Response> {
	const url = resolvePath(path);

	const send = () => {
		const headers = new Headers(options?.headers);
		if (body !== undefined) headers.set('Content-Type', 'application/json');
		const bearer = authenticator?.header();
		if (bearer) headers.set('Authorization', bearer);

		return fetchFn(url, {
			...options,
			method,
			headers,
			body: body === undefined ? undefined : JSON.stringify(body)
		});
	};

	const response = await send();
	if (response.status !== 401 || !authenticator || path.startsWith(SESSION_PROTOCOL)) {
		return response;
	}

	const retried = (await authenticator.renew()) ? await send() : response;
	if (retried.status === 401) authenticator.expired();
	return retried;
}

export async function get(
	path: Path,
	options?: RequestInit,
	fetchFn: typeof fetch = fetch
): Promise<Response> {
	let response: Response;
	try {
		response = await request('GET', path, undefined, options, fetchFn);
	} catch (err) {
		// Network errors (CORS, timeout, connection failed) don't have status codes
		console.error('Network error:', err);
		throw error(503, `Network error: Unable to reach server`);
	}

	if (!response.ok) {
		throw error(response.status, response.statusText || 'Request failed');
	}

	return response;
}

export function post(
	path: Path,
	body: unknown,
	options?: RequestInit,
	fetchFn: typeof fetch = fetch
) {
	return request('POST', path, body, options, fetchFn);
}

export function put(
	path: Path,
	body?: unknown,
	options?: RequestInit,
	fetchFn: typeof fetch = fetch
) {
	return request('PUT', path, body, options, fetchFn);
}

export function patch(
	path: Path,
	body: unknown,
	options?: RequestInit,
	fetchFn: typeof fetch = fetch
) {
	return request('PATCH', path, body, options, fetchFn);
}

export function del(
	path: Path,
	body?: unknown,
	options?: RequestInit,
	fetchFn: typeof fetch = fetch
) {
	return request('DELETE', path, body, options, fetchFn);
}
