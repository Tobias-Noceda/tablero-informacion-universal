import type { PostIt, UUID } from "$types/api";

import * as api from "$modules/api.svelte"

// TODO: support custom post its
export async function create_custom(board: UUID) {
    const res = await api.post("/v1/post-its", { board });
    return await res.json() as PostIt;
}

export async function create_well_known(board: UUID, well_known: string, params: Record<string, string>) {
    const res = await api.post("/v1/post-its", { board, well_known, params });
    return await res.json() as PostIt;
}

export async function del(id: UUID) {
    await api.del(`/v1/post-its/${id}`);
}

export async function execute(id: UUID) {
    const res = await api.get(`/v1/post-its/${id}`);
    return await res.json() as Record<string, string>;
}

export async function get_settings(id: UUID) {
    const res = await api.get(`/v1/post-its/${id}/settings`);
    return await res.json() as PostIt;
}

// TODO
export async function update_settings(id: UUID, params: Record<string, string>) {
    await api.patch(`/v1/post-its/${id}/settings`, { params });
}

export async function move(id: UUID, x: number, y: number) {
    await api.patch(`/v1/post-its/${id}/position`, { x, y });
}
