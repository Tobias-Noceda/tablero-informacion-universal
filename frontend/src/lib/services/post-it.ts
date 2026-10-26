import type { PostIt, PostItSettings, SecretRef, Strand, UUID } from "$types/api";

import * as api from "$modules/api.svelte"
import { mapPostitWithTitle, type PostItWithTitle } from "$lib/helpers/post-it";

// TODO: support custom post its
export async function create_custom(board: UUID) {
    const res = await api.post("/v1/post-its", { board });
    return await res.json() as PostIt;
}

export async function create_well_known(
    board: UUID,
    title: { text: string, vars: boolean },
    well_known: string,
    params: Record<string, string>,
    bindings: Record<string, SecretRef> = {},
) {
    const res = await api.post("/v1/post-its", { board, title, well_known, params, bindings });
    return mapPostitWithTitle(await res.json() as PostItWithTitle);
}

export async function update_well_known_title(
    postItId: UUID,
    title: { text: string, vars: boolean },
) {
    await api.patch(`/v1/post-its/${postItId}/title`, { title });
}

export async function update_well_known_settings(
    postItId: UUID,
    params: Record<string, string>,
    bindings: Record<string, SecretRef> = {},
) {
    await api.patch(`/v1/post-its/${postItId}/settings`, { params, bindings });
}

export async function del(id: UUID): Promise<Strand[]> {
    const deletedEdges = await api.del(`/v1/post-its/${id}`).then(async (res) => await res.json() as Strand[]);
    return deletedEdges;
}

// `fresh` skips every cache on the way, so the card runs again now.
export async function execute(id: UUID, { fresh = false }: { fresh?: boolean } = {}) {
    const res = fresh
        ? await api.get(`/v1/post-its/${id}?fresh=1`, { cache: "no-store" })
        : await api.get(`/v1/post-its/${id}`);
    return await res.json() as Record<string, string>;
}

export async function get_settings(id: UUID) {
    const res = await api.get(`/v1/post-its/${id}/settings`);
    return await res.json() as PostItSettings;
}

export async function update_settings(
    id: UUID,
    params: Record<string, string>,
    bindings?: Record<string, SecretRef>,
) {
    await api.patch(`/v1/post-its/${id}/settings`, { params, bindings });
}

export async function move(id: UUID, x: number, y: number) {
    await api.patch(`/v1/post-its/${id}/position`, { x, y });
}
