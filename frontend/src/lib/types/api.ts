export type UUID = string;

export type Envs = {
    name: string;
    // References to AWS Secret Manager ("type:id:name")
    key: `${'global' | 'user' | 'board' | 'post-it'}:${UUID}:${string}`
};

// export type User = {
//     cognito_id: string;
//     boards: Board['id'][]
//     // Future ideas:
//     // plan: string;
//     // team: UUID;
//     // idk, i forgot
// }

export type Board = {
    id: UUID;
    name: string;
    owner: string; // cognito_id
    collaborators: string[]; // cognito_ids
    postits: {
        id: UUID;
        type?: string;
        position: { x: number; y: number; };
        // data: { url: `/api/post-its/${UUID}` };
    }[];
    strands: {
        source: string;
        target: string;
        // DTO only:
        // id: `${this.source}-${this.target}`;
    }[];
    // Board defined env variables
    envs: Envs[];
};

export type PostIt = {
    id: UUID;
    params: Map<string, string>;

    // Used to reference "propietary" post-its, which may reference
    // global secrets.
    well_known?: string;

    // If well-known, these options are NOT in the user's control,
    // they are persisted in the DB, but completely hidden in the DTO.
    resource: URL;
    request: Pick<RequestInit, 'method' | 'headers' | 'body'>;
    response: 'json';
    // https://github.com/itchyny/gojq
    query: string;
    rate?: number; // A rate-less post-it should only be updated on creation
    // Board + Post-it defined env variables
    envs: Envs[];

    // Whose credentials the card runs with: whoever last saved its params
    // or bindings. Empty on cards from before this existed (board owner).
    run_as?: string;
    // "$TOKEN"s the card uses that do not live in the board's own scope,
    // keyed by the token name without the dollar sign.
    bindings?: Record<string, SecretRef>;
}

export type Strand = {
    id: UUID;
    source: UUID;
    target: UUID;
}

export type SecretKind = 'api_key' | 'bearer' | 'basic' | 'oauth2';

export type OAuth2Flow = 'client_credentials' | 'authorization_code';

// A service the platform has registered its own application with. Users
// consent to it instead of bringing their own client.
export type OAuthProvider = 'google' | 'discord';

export type OAuthProviderStatus = {
    provider: OAuthProvider;
    configured: boolean;
}

export type ScopeKind = 'board' | 'member' | 'user' | 'group' | 'system';

// Who a secret belongs to. member owners are "<board id>:<user id>".
export type SecretScope = {
    kind: ScopeKind;
    owner?: string;
}

export type SecretRef = {
    scope: SecretScope;
    name: string;
}

export type AudienceKind = 'user' | 'group' | 'board';

export type Audience = {
    kind: AudienceKind;
    id: string;
}

// Lets an audience outside the scope bind the secret, everywhere or on one board.
export type Grant = {
    to: Audience;
    board?: UUID;
}

// What listing a scope's secrets returns. The value is never part of it.
export type SecretMeta = {
    scope: SecretScope;
    name: string;
    kind: SecretKind;
    grants: Grant[];
    // Only present for OAuth2 credentials.
    flow?: OAuth2Flow;
    authorized: boolean;
    // Set when the grant was obtained with the platform's application.
    provider?: OAuthProvider;
    created_at: string;
    updated_at: string;
}

export type OAuth2Config = {
    name: string;
    flow: OAuth2Flow;
    client_id: string;
    client_secret: string;
    token_url: string;
    auth_url?: string;
    scopes?: string;
}
