import type { Board, PostIt, UUID } from "$types/api";

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

export async function share(id: UUID, user: UUID) {
    await api.post(`/v1/boards/${id}/collaborators`, { user });
}

export async function unshare(id: UUID, user: UUID) {
    await api.del(`/v1/boards/${id}/collaborators`, { user });
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
