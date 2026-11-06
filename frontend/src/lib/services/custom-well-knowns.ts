import * as api from '$modules/api.svelte';
import type {
	BoardWellKnown,
	CustomWellKnown,
	CustomWellKnownBody,
	CustomWellKnownScope,
	UUID
} from '$types/api';

// Why a change failed, as the backend's `error` code (invalid_well_known,
// not_found, limit_reached); `detail` explains an invalid one.
export class CustomWellKnownError extends Error {
	constructor(
		readonly code: string,
		readonly detail?: string
	) {
		super(detail ? `${code}: ${detail}` : code);
		this.name = 'CustomWellKnownError';
	}
}

async function ensure(response: Response) {
	if (response.ok) return response;
	const body = await response.json().catch(() => ({}));
	throw new CustomWellKnownError(
		typeof body?.error === 'string' ? body.error : 'unknown',
		typeof body?.message === 'string' ? body.message : undefined
	);
}

export function userScope(user: string): CustomWellKnownScope {
	return { kind: 'user', owner: user };
}

export function boardScope(board: UUID): CustomWellKnownScope {
	return { kind: 'board', owner: board };
}

export function orgScope(org: UUID): CustomWellKnownScope {
	return { kind: 'org', owner: org };
}

// The route prefix that lists and creates a scope's card types.
export function pathOf(scope: CustomWellKnownScope): `/${string}` {
	switch (scope.kind) {
		case 'user':
			return `/v1/users/${encodeURIComponent(scope.owner)}/well-knowns`;
		case 'board':
			return `/v1/boards/${scope.owner}/well-knowns`;
		case 'org':
			return `/v1/orgs/${scope.owner}/well-knowns`;
	}
}

const itemPath = (key: string) => `/v1/custom-well-knowns/${encodeURIComponent(key)}` as const;

export async function list(scope: CustomWellKnownScope) {
	const response = await ensure(await api.request('GET', pathOf(scope)));
	return (await response.json()) as CustomWellKnown[];
}

// Everything a board needs: the types the caller may drop on it (usable) and
// those its cards were made from.
export async function forBoard(board: UUID) {
	const response = await ensure(await api.request('GET', `/v1/boards/${board}/well-knowns/usable`));
	return (await response.json()) as BoardWellKnown[];
}

export async function create(scope: CustomWellKnownScope, body: CustomWellKnownBody) {
	const response = await ensure(await api.post(pathOf(scope), body));
	return (await response.json()) as CustomWellKnown;
}

// Cards already made keep the definition they were created with.
export async function update(key: string, body: CustomWellKnownBody) {
	const response = await ensure(await api.put(itemPath(key), body));
	return (await response.json()) as CustomWellKnown;
}

// For good: cards made from it keep running but can no longer be edited.
export async function remove(key: string) {
	await ensure(await api.del(itemPath(key)));
}
