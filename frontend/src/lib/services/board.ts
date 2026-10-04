import type { Board, BoardMember, BoardMemberSummary, PostIt, UUID } from "$types/api";

import * as api from "$modules/api.svelte"

export async function create(name: string) {
    const res = await api.post("/v1/boards", { name });
    return await res.json() as Board;
}

// Every board the signed-in user owns or collaborates on.
export async function get_all() {
    const res = await api.get("/v1/boards");
    return await res.json() as Board[];
}

export async function get(id: UUID) {
    const res = await api.get(`/v1/boards/${id}`);
    return await res.json() as Board;
}

export async function del(id: UUID) {
    await api.del(`/v1/boards/${id}`);
}

export async function get_post_its(id: UUID) {
    const res = await api.get(`/v1/boards/${id}/post-its`);
    return await res.json() as PostIt[];
}

// Why sharing failed, as the backend's `error` code (user_not_found,
// invalid_role, owner_role...).
export class ShareError extends Error {
    constructor(readonly code: string) {
        super(code);
        this.name = "ShareError";
    }
}

async function failed(res: Response) {
    const body = await res.json().catch(() => ({}));
    return new ShareError(typeof body?.error === "string" ? body.error : "unknown");
}

export async function members(id: UUID) {
    const res = await api.get(`/v1/boards/${id}/members`);
    return await res.json() as BoardMemberSummary[];
}

// Adds whoever registered with email, or changes their role.
export async function setMember(id: UUID, email: string, role: BoardMember["role"]) {
    const res = await api.put(`/v1/boards/${id}/members`, { email, role });
    if (!res.ok) throw await failed(res);
    return await res.json() as BoardMemberSummary;
}

// The owner removes someone, or anyone removes themselves (leaves).
export async function removeMember(id: UUID, user: UUID) {
    const res = await api.del(`/v1/boards/${id}/members/${user}`);
    if (!res.ok) throw await failed(res);
}

export async function rename(id: UUID, name: string) {
    await api.patch(`/v1/boards/${id}/name`, { name });
}

export async function online(id: UUID, peer: UUID) {
	const res = await api.put(`/v1/boards/${id}/online?${new URLSearchParams({ peer })}`);
    return await res.json() as string[];
}

export async function offline(id: UUID, peer: UUID) {
	return api.del(`/v1/boards/${id}/online?${new URLSearchParams({ peer })}`, undefined, { keepalive: true });
}
