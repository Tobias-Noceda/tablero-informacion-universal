import * as api from '$modules/api.svelte';
import type { Org, OrgDetail, OrgMemberSummary, OrgRole, UUID } from '$types/api';

// Why a change failed, as the backend's `error` code (user_not_found,
// invalid_role, last_admin, org_not_empty...).
export class OrgError extends Error {
	constructor(readonly code: string) {
		super(code);
		this.name = 'OrgError';
	}
}

async function ensure(response: Response) {
	if (response.ok) return response;
	const body = await response.json().catch(() => ({}));
	throw new OrgError(typeof body?.error === 'string' ? body.error : 'unknown');
}

// The organizations the signed-in user is in, each with their role.
export async function list() {
	const response = await api.get('/v1/orgs');
	return (await response.json()) as Org[];
}

export async function get(id: UUID) {
	const response = await api.get(`/v1/orgs/${id}`);
	return (await response.json()) as OrgDetail;
}

// The caller becomes its first admin.
export async function create(name: string) {
	const response = await ensure(await api.post('/v1/orgs', { name }));
	return (await response.json()) as Org;
}

export async function rename(id: UUID, name: string) {
	await ensure(await api.patch(`/v1/orgs/${id}`, { name }));
}

// Refused with org_not_empty while a board or a group still belongs to it.
export async function del(id: UUID) {
	await ensure(await api.del(`/v1/orgs/${id}`));
}

// Adds whoever registered with email, or changes their role.
export async function setMember(id: UUID, email: string, role: OrgRole) {
	const response = await ensure(await api.put(`/v1/orgs/${id}/members`, { email, role }));
	return (await response.json()) as OrgMemberSummary;
}

// An admin removes someone, or anyone removes themselves (leaves).
export async function removeMember(id: UUID, user: UUID) {
	await ensure(await api.del(`/v1/orgs/${id}/members/${user}`));
}
