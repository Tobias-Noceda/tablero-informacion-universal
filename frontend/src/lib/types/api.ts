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

export type Strand = {
    id: UUID;
    source: UUID;
    target: UUID;
}
