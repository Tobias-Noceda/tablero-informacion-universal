import * as api from '$modules/api.svelte';
import type { WellKnown } from '$types/api';

// Why a change failed, as the backend's `error` code (invalid_well_known,
// forbidden, not_found, not_deleted); `message` explains an invalid one.
export class WellKnownError extends Error {
	constructor(
		readonly code: string,
		readonly detail?: string
	) {
		super(detail ? `${code}: ${detail}` : code);
		this.name = 'WellKnownError';
	}
}

async function ensure(response: Response) {
	if (response.ok) return response;
	const body = await response.json().catch(() => ({}));
	throw new WellKnownError(
		typeof body?.error === 'string' ? body.error : 'unknown',
		typeof body?.message === 'string' ? body.message : undefined
	);
}

// The cards anyone signed in may create, in picker order.
export async function list() {
	const response = await api.get('/v1/well-knowns');
	return (await response.json()) as WellKnown[];
}

export async function get(key: string) {
	const response = await api.get(`/v1/well-knowns/${encodeURIComponent(key)}`);
	return (await response.json()) as WellKnown;
}

// Admins only: creates or replaces the definition (reviving a deleted one).
export async function put(key: string, definition: Omit<WellKnown, 'key' | 'builtin'>) {
	const response = await ensure(await api.put(`/v1/well-knowns/${encodeURIComponent(key)}`, definition));
	return (await response.json()) as WellKnown;
}

// Admins only: hides it from the picker; cards already made keep running.
export async function remove(key: string) {
	await ensure(await api.del(`/v1/well-knowns/${encodeURIComponent(key)}`));
}

// Admins only: brings a deleted one back as it was.
export async function restore(key: string) {
	const response = await ensure(await api.post(`/v1/well-knowns/${encodeURIComponent(key)}/restore`, {}));
	return (await response.json()) as WellKnown;
}
