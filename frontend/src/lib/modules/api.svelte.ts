import { error } from '@sveltejs/kit';
import { SvelteURL } from 'svelte/reactivity';

function resolvePath(path: `/${string}`) {
	try {
		// Check if the path is a valid URL
		return new SvelteURL(path);
	} catch (e) {
		if (!(e instanceof TypeError)) throw e;
		// Assume it's a relative path (/api/...)
		return new SvelteURL(`http://localhost:31126${path}`);
	}
}

export async function get(path: `/${string}`, options?: RequestInit, fetchFn: typeof fetch = fetch): Promise<Response> {
	const url = resolvePath(path);

	let response: Response;
	try {
		response = await fetchFn(new SvelteURL(url), options);
	} catch (err) {
		// Network errors (CORS, timeout, connection failed) don't have status codes
		console.error('Network error:', err);
		throw error(503, `Network error: Unable to reach server`);
	}

	if (!response.ok) {
		const message = response.statusText || 'Request failed';
		throw error(response.status, message);
	}

	return response;
}

export function post(path: `/${string}`, body: unknown, options?: RequestInit, fetchFn: typeof fetch = fetch): Promise<Response> {
	const url = resolvePath(path);

	return fetchFn(new SvelteURL(url), {
		...options,
		method: 'POST',
		headers: {
			...options?.headers,
			'Content-Type': 'application/json',
		},
		// Don't stringify FormData
		body: JSON.stringify(body)
	});
};

export function put(path: `/${string}`, body: unknown, options?: RequestInit, fetchFn: typeof fetch = fetch): Promise<Response> {
	const url = resolvePath(path);

	return fetchFn(new SvelteURL(url), {
		...options,
		method: 'PUT',
		headers: {
			...options?.headers,
			'Content-Type': 'application/json',
		},
		body: JSON.stringify(body)
	});
};

export function patch(path: `/${string}`, body: unknown, options?: RequestInit, fetchFn: typeof fetch = fetch): Promise<Response> {
	return fetchFn(new SvelteURL(path), {
		...options,
		method: 'PATCH',
		headers: {
			...options?.headers,
			'Content-Type': 'application/json',
		},
		body: JSON.stringify(body)
	});
};

export function del(path: `/${string}`, options?: RequestInit, fetchFn: typeof fetch = fetch): Promise<Response> {
	return fetchFn(new SvelteURL(path), {
		...options,
		method: 'DELETE',
		headers: {
			...options?.headers,
		}
	});
};
