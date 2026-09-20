import type { PostIt, Strand, UUID } from "$types/api";

import * as api from "$modules/api.svelte"

const DEFAULT_COGNITO_ID = "Messi";

// TODO: support custom post its
export async function create_custom(board: UUID, cognito_id = DEFAULT_COGNITO_ID) {
    const res = await api.post("/v1/post-its", { cognito_id, board });
    return await res.json() as PostIt;
}

export async function create_well_known(board: UUID, well_known: string, params: Record<string, string>, cognito_id = DEFAULT_COGNITO_ID) {
    const res = await api.post("/v1/post-its", { cognito_id, board, well_known, params });
    return await res.json() as PostIt;
}

export async function del(id: UUID): Promise<Strand[]> {
    const deletedEdges = await api.del(`/v1/post-its/${id}`).then(async (res) => await res.json() as Strand[]);
    return deletedEdges;
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
export async function update_settings(id: UUID, params: Record<string, string>, cognito_id = DEFAULT_COGNITO_ID) {
    await api.patch(`/v1/post-its/${id}/settings`, { cognito_id, params });
}

export async function move(id: UUID, x: number, y: number) {
    await api.patch(`/v1/post-its/${id}/position`, { x, y });
}
