import type {
    OAuth2Config,
    OAuthProvider,
    OAuthProviderStatus,
    SecretKind,
    SecretMeta,
    SecretScope,
    UUID,
} from "$types/api";

import * as api from "$modules/api.svelte"
import { CURRENT_USER } from "$modules/api.svelte";

export function boardScope(board: UUID): SecretScope {
    return { kind: "board", owner: board };
}

// What one user keeps on one board for themselves.
export function memberScope(board: UUID, user: string): SecretScope {
    return { kind: "member", owner: `${board}:${user}` };
}

// The route prefix that serves a scope's secrets.
export function pathOf(scope: SecretScope): `/${string}` {
    switch (scope.kind) {
        case "board":
            return `/v1/boards/${scope.owner}`;
        case "member": {
            const [board, user] = splitMember(scope.owner ?? "");
            return `/v1/boards/${board}/members/${encodeURIComponent(user)}`;
        }
        case "user":
            return `/v1/users/${encodeURIComponent(scope.owner ?? "")}`;
        case "group":
            return `/v1/groups/${scope.owner}`;
        case "system":
            return "/v1/system";
    }
}

function splitMember(owner: string): [string, string] {
    const separator = owner.indexOf(":");
    return [owner.slice(0, separator), owner.slice(separator + 1)];
}

export async function list(scope: SecretScope, cognito_id = CURRENT_USER) {
    const res = await api.get(`${pathOf(scope)}/secrets?${new URLSearchParams({ cognito_id })}`);
    return await res.json() as SecretMeta[];
}

// Everything the caller may bind into a card on this board.
export async function usable(board: UUID, cognito_id = CURRENT_USER) {
    const res = await api.get(`/v1/boards/${board}/secrets/usable?${new URLSearchParams({ cognito_id })}`);
    return await res.json() as SecretMeta[];
}

export async function put(
    scope: SecretScope,
    name: string,
    kind: SecretKind,
    value: string,
    cognito_id = CURRENT_USER,
) {
    await api.put(`${pathOf(scope)}/secrets`, { cognito_id, name, kind, value });
}

export async function del(scope: SecretScope, name: string, cognito_id = CURRENT_USER) {
    await api.del(`${pathOf(scope)}/secrets/${encodeURIComponent(name)}?${new URLSearchParams({ cognito_id })}`);
}

export async function put_oauth2(scope: SecretScope, config: OAuth2Config, cognito_id = CURRENT_USER) {
    await api.put(`${pathOf(scope)}/oauth2`, { cognito_id, ...config });
}

export async function providers() {
    const res = await api.get(`/v1/oauth2/providers`);
    return await res.json() as OAuthProviderStatus[];
}

export async function connect(
    scope: SecretScope,
    provider: OAuthProvider,
    name: string,
    redirect_uri: string,
    cognito_id = CURRENT_USER,
) {
    const res = await api.post(`${pathOf(scope)}/oauth2/connect`, { cognito_id, provider, name, redirect_uri });
    const { authorization_url } = await res.json() as { authorization_url: string };
    return authorization_url;
}

export async function authorize(
    scope: SecretScope,
    name: string,
    redirect_uri: string,
    cognito_id = CURRENT_USER,
) {
    const query = new URLSearchParams({ cognito_id, name, redirect_uri });
    const res = await api.get(`${pathOf(scope)}/oauth2/authorize?${query}`);
    const { authorization_url } = await res.json() as { authorization_url: string };
    return authorization_url;
}
