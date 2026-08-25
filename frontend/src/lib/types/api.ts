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
}

export type SecretKind = 'api_key' | 'bearer' | 'basic' | 'oauth2';

export type OAuth2Flow = 'client_credentials' | 'authorization_code';

// What listing a board's secrets returns. The value is never part of it.
export type SecretMeta = {
    name: string;
    kind: SecretKind;
    // Only present for OAuth2 credentials.
    flow?: OAuth2Flow;
    authorized: boolean;
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
