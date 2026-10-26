import type { BoardPostIt, PostItData } from "../types/api";

import * as api from "./api";

import Mustache from "mustache";

// A card's data: what its query produced (GET /post-its/:id).
export async function dataOf(id: string) {
    const res = await api.get(`/v1/post-its/${id}`);
    return (await res.json()) as PostItData;
}

// A card's title as the user sees it: a Mustache template over its data
// when `vars`, the raw text otherwise or when the data can't be fetched.
export async function titleOf(postit: BoardPostIt, data?: PostItData) {
    if (!postit.title.vars) return postit.title.text;
    try {
        return Mustache.render(
            postit.title.text,
            data ?? (await dataOf(postit.id)),
        );
    } catch {
        return postit.title.text;
    }
}

// One memory per line: `id - (x; y) - title`.
export async function lines(postits: BoardPostIt[]) {
    const titles = await Promise.all(postits.map((p) => titleOf(p)));
    return postits.map(
        ({ id, position }, i) =>
            `${id} - (${position.x}; ${position.y}) - ${titles[i]}`,
    );
}
