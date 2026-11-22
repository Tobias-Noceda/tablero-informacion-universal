import type { CardStyle, OneOffRequest, PostItSettings, SecretRef, Strand, UUID } from "$types/api";

import * as api from "$modules/api.svelte"
import { mapPostitWithTitle, type PostItWithTitle } from "$lib/helpers/post-it";

// A refused card change, with the backend's `error` text.
export class PostItError extends Error {
    constructor(readonly status: number, message: string) {
        super(message);
        this.name = 'PostItError';
    }
}

async function ensure(response: Response) {
    if (response.ok) return response;
    const body = await response.json().catch(() => ({}));
    throw new PostItError(response.status, typeof body?.error === 'string' ? body.error : response.statusText);
}

// A card with its own request, without a card type.
export async function create_one_off(
    board: UUID,
    title: { text: string, vars: boolean },
    body: OneOffRequest,
) {
    const res = await ensure(await api.post("/v1/post-its", { board, title, ...body }));
    return mapPostitWithTitle(await res.json() as PostItWithTitle);
}

export async function update_one_off(postItId: UUID, body: OneOffRequest) {
    await ensure(await api.patch(`/v1/post-its/${postItId}/settings`, body));
}

export async function create_well_known(
    board: UUID,
    title: { text: string, vars: boolean },
    well_known: string,
    params: Record<string, string>,
    bindings: Record<string, SecretRef> = {},
) {
    const res = await ensure(await api.post("/v1/post-its", { board, title, well_known, params, bindings }));
    return mapPostitWithTitle(await res.json() as PostItWithTitle);
}

export async function update_well_known_title(
    postItId: UUID,
    title: { text: string, vars: boolean },
) {
    await ensure(await api.patch(`/v1/post-its/${postItId}/title`, { title }));
}

export async function update_well_known_settings(
    postItId: UUID,
    params: Record<string, string>,
    bindings: Record<string, SecretRef> = {},
) {
    await ensure(await api.patch(`/v1/post-its/${postItId}/settings`, { params, bindings }));
}

export async function del(id: UUID): Promise<Strand[]> {
    const deletedEdges = await api.del(`/v1/post-its/${id}`).then(async (res) => await res.json() as Strand[]);
    return deletedEdges;
}

// Why a card could not run: the HTTP status (0 when the request never got an
// answer), the backend's `code` (source_status, source_timeout, ...) and, for
// source_status, what the card's source answered.
export class CardError extends Error {
    constructor(
        readonly status: number,
        readonly code?: string,
        readonly sourceStatus?: number,
        message = '',
    ) {
        super(message);
        this.name = 'CardError';
    }
}

// `fresh` skips every cache on the way, so the card runs again now; only a
// plan with manual refresh may ask for it (402 otherwise). `reload` only
// skips the browser's copy: after an edit the backend has already dropped
// its own, so a plain request runs the card again on any plan.
export async function execute(
    id: UUID,
    { fresh = false, reload = false }: { fresh?: boolean; reload?: boolean } = {},
) {
    const path: `/${string}` = fresh ? `/v1/post-its/${id}?fresh=1` : `/v1/post-its/${id}`;
    let res: Response;
    try {
        res = await api.request('GET', path, undefined, fresh || reload ? { cache: 'no-store' } : undefined);
    } catch {
        throw new CardError(0, 'network');
    }
    if (!res.ok) throw await cardError(res);
    return await res.json() as Record<string, string>;
}

async function cardError(res: Response) {
    const body = await res.json().catch(() => ({}));
    return new CardError(
        res.status,
        typeof body?.code === 'string' ? body.code : undefined,
        typeof body?.status === 'number' ? body.status : undefined,
        typeof body?.error === 'string' ? body.error : res.statusText,
    );
}

// What a one-off card being written would show, without saving it: its
// outputs, or the whole JSON response when it has none. Fails like a card
// (CardError); 429 when the editor tried too often.
export async function preview(
    board: UUID,
    body: Pick<OneOffRequest, 'resource' | 'request' | 'response' | 'bindings'> & { query?: Record<string, string> },
): Promise<{ outputs?: Record<string, unknown>; response?: unknown }> {
    let res: Response;
    try {
        res = await api.request('POST', '/v1/post-its/preview', { board, ...body });
    } catch {
        throw new CardError(0, 'network');
    }
    if (!res.ok) throw await cardError(res);
    return await res.json();
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
    await ensure(await api.patch(`/v1/post-its/${id}/position`, { x, y }));
}

// Changes only the fields sent; an empty value (0 for a size) goes back to
// the default. Answers the card's resulting style.
export async function style(id: UUID, patch: CardStyle): Promise<CardStyle> {
    const res = await ensure(await api.patch(`/v1/post-its/${id}/style`, patch));
    return await res.json() as CardStyle;
}
