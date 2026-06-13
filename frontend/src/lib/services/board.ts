import type { Board, PostIt, UUID } from "$types/api";

import * as api from "$modules/api.svelte"

export async function create(name: string, owner = "Messi") {
    const res = await api.post("/v1/boards", { name, owner });
    return await res.json() as Board;
}

export async function get_all(cognito_id = "Messi") {
    const res = await api.get(`/v1/boards?${new URLSearchParams({ cognito_id })}`);
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

export async function share(id: UUID, cognito_id = "Messi") {
    await api.post(`/v1/boards/${id}/collaborators`, { cognito_id });
}

// TODO: broken
export async function unshare(id: UUID, cognito_id = "Messi") {
    await api.del(`/v1/boards/${id}/collaborators?${new URLSearchParams({ cognito_id })}`);
}

export async function connect(id: UUID, source: UUID, target: UUID) {
	await api.post(`/api/v1/boards/${id}/strands`, { source, target });
}

// TODO: broken
export async function disconnect(id: UUID, source: UUID, target: UUID) {
	await api.del(`/api/v1/boards/${id}/strands`);
}

export async function rename(id: UUID, name: string) {
    await api.post(`/v1/boards/${id}/name`, { name });
}
