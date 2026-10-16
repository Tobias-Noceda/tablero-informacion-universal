export type UUID = string;

export type Envs = {
    name: string;
    // References to AWS Secret Manager ("type:id:name")
    key: `${"global" | "user" | "board" | "post-it"}:${UUID}:${string}`;
};

export type Paginated<T> = {
    _links: {
        first?: string;
        prev?: string;
        next?: string;
        last?: string;
    };
    _pageInfo?: {
        currentPage?: number;
        totalPages?: number;
        currentDate?: Date;
        maxDate?: Date;
    };
    results: T[];
};

export type PostIt = {
    id: UUID;
    type?: string;
    position: { x: number; y: number };
    data: Record<string, unknown>;
};

export type Board = {
    id: UUID;
    name: string;
    owner: string; // cognito_id
    collaborators: string[]; // cognito_ids
    postits: {
        id: UUID;
        title: {
            text: string;
            vars: boolean;
        };
        type?: string;
        position: { x: number; y: number };
    }[];
    strands: {
        source: string;
        target: string;
    }[];
    // Board defined env variables
    envs: Envs[];
};

export type Strand = {
    id: UUID;
    source: UUID;
    target: UUID;
};
