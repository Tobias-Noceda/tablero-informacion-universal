import type { AccessToken, CreatedAccessToken, UUID } from "$types/api";

import * as api from "$modules/api.svelte"

// The lifetimes the backend accepts, in days.
export const LIFETIMES = ['7', '15', '30'] as const;
export type Lifetime = (typeof LIFETIMES)[number];

// Why a token call failed, as the backend's `error` code (invalid_name,
// invalid_expiry, too_many_tokens, rate_limited...).
export class TokenError extends Error {
    constructor(readonly code: string) {
        super(code);
        this.name = "TokenError";
    }
}

async function failed(res: Response) {
    const body = await res.json().catch(() => ({}));
    return new TokenError(typeof body?.error === "string" ? body.error : "unknown");
}

export async function list(user: UUID) {
    const res = await api.get(`/v1/users/${user}/tokens`);
    return await res.json() as AccessToken[];
}

export async function create(user: UUID, name: string, days: Lifetime) {
    const res = await api.post(`/v1/users/${user}/tokens`, { name, days: Number(days) });
    if (!res.ok) throw await failed(res);
    return await res.json() as CreatedAccessToken;
}

export async function revoke(user: UUID, id: UUID) {
    const res = await api.del(`/v1/users/${user}/tokens/${id}`);
    if (!res.ok) throw await failed(res);
}
