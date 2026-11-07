import * as api from '$modules/api.svelte';
import type {
	BoardPostitTemplate,
	PostitTemplate,
	PostitTemplateBody,
	PostitTemplateScope,
	UUID
} from '$types/api';

// Why a change failed, as the backend's `error` code (invalid_postit_template,
// not_found, limit_reached); `detail` explains an invalid one.
export class PostitTemplateError extends Error {
	constructor(
		readonly code: string,
		readonly detail?: string
	) {
		super(detail ? `${code}: ${detail}` : code);
		this.name = 'PostitTemplateError';
	}
}

async function ensure(response: Response) {
	if (response.ok) return response;
	const body = await response.json().catch(() => ({}));
	throw new PostitTemplateError(
		typeof body?.error === 'string' ? body.error : 'unknown',
		typeof body?.message === 'string' ? body.message : undefined
	);
}

export function userScope(user: string): PostitTemplateScope {
	return { kind: 'user', owner: user };
}

export function boardScope(board: UUID): PostitTemplateScope {
	return { kind: 'board', owner: board };
}

export function orgScope(org: UUID): PostitTemplateScope {
	return { kind: 'org', owner: org };
}

// The route prefix that lists and creates a scope's card types.
export function pathOf(scope: PostitTemplateScope): `/${string}` {
	switch (scope.kind) {
		case 'user':
			return `/v1/users/${encodeURIComponent(scope.owner)}/postit-templates`;
		case 'board':
			return `/v1/boards/${scope.owner}/postit-templates`;
		case 'org':
			return `/v1/orgs/${scope.owner}/postit-templates`;
	}
}

const itemPath = (key: string) => `/v1/postit-templates/${encodeURIComponent(key)}` as const;

export async function list(scope: PostitTemplateScope) {
	const response = await ensure(await api.request('GET', pathOf(scope)));
	return (await response.json()) as PostitTemplate[];
}

// Everything a board needs: the types the caller may drop on it (usable) and
// those its cards were made from.
export async function forBoard(board: UUID) {
	const response = await ensure(
		await api.request('GET', `/v1/boards/${board}/postit-templates/usable`)
	);
	return (await response.json()) as BoardPostitTemplate[];
}

export async function create(scope: PostitTemplateScope, body: PostitTemplateBody) {
	const response = await ensure(await api.post(pathOf(scope), body));
	return (await response.json()) as PostitTemplate;
}

// Cards already made keep the definition they were created with.
export async function update(key: string, body: PostitTemplateBody) {
	const response = await ensure(await api.put(itemPath(key), body));
	return (await response.json()) as PostitTemplate;
}

// For good: cards made from it keep running but can no longer be edited.
export async function remove(key: string) {
	await ensure(await api.del(itemPath(key)));
}
