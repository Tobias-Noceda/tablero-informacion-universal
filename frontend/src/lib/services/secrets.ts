import type { OAuth2Config, SecretKind, SecretMeta, UUID } from "$types/api";

import * as api from "$modules/api.svelte"

const DEFAULT_COGNITO_ID = "Messi";

export async function list(board: UUID, cognito_id = DEFAULT_COGNITO_ID) {
    const res = await api.get(`/v1/boards/${board}/secrets?${new URLSearchParams({ cognito_id })}`);
    return await res.json() as SecretMeta[];
}

export async function put(
    board: UUID,
    name: string,
    kind: SecretKind,
    value: string,
    cognito_id = DEFAULT_COGNITO_ID,
) {
    await api.put(`/v1/boards/${board}/secrets`, { cognito_id, name, kind, value });
}

export async function del(board: UUID, name: string, cognito_id = DEFAULT_COGNITO_ID) {
    await api.del(`/v1/boards/${board}/secrets/${encodeURIComponent(name)}?${new URLSearchParams({ cognito_id })}`);
}

export async function put_oauth2(board: UUID, config: OAuth2Config, cognito_id = DEFAULT_COGNITO_ID) {
    await api.put(`/v1/boards/${board}/oauth2`, { cognito_id, ...config });
}

export async function authorize(
    board: UUID,
    name: string,
    redirect_uri: string,
    cognito_id = DEFAULT_COGNITO_ID,
) {
    const query = new URLSearchParams({ cognito_id, name, redirect_uri });
    const res = await api.get(`/v1/boards/${board}/oauth2/authorize?${query}`);
    const { authorization_url } = await res.json() as { authorization_url: string };
    return authorization_url;
}
