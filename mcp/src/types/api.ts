// Mirrors of the backend models (backend/common/models) the tools read.
export type UUID = string;

export type BoardRole = "owner" | "editor" | "viewer";

export type BoardMember = {
    user: UUID;
    role: BoardRole;
};

export type BoardPostIt = {
    id: UUID;
    type: string;
    title: {
        text: string;
        // Whether text is a Mustache template over the card's data.
        vars: boolean;
    };
    position: { x: number; y: number };
};

// Which ends of a strand point at their memory.
export type StrandArrow = "none" | "start" | "end" | "both";

export type Strand = {
    id: UUID;
    source: UUID;
    target: UUID;
    // Optional style; the tools only read and write these two.
    label?: string;
    arrow?: StrandArrow;
};

export type Board = {
    id: UUID;
    name: string;
    owner: UUID;
    members: BoardMember[];
    org?: UUID;
    postits: BoardPostIt[];
    strands: Strand[];
    // The caller's role on the board.
    role?: BoardRole;
};

// GET /orgs: only what the tools read.
export type Org = {
    id: UUID;
    name: string;
};

// What a card's query produced: GET /post-its/:id.
export type PostItData = Record<string, unknown>;

// GET /post-its/:id/settings: how a card reads, credentials only as "$NAME".
export type PostItSettings = {
    id: UUID;
    wellknown?: string;
    rate: number;
    resource?: string | null;
    request?: {
        method?: string;
        headers?: Record<string, string> | null;
        queries?: Record<string, string> | null;
        body?: string;
    };
    response?: string;
    query?: Record<string, string> | null;
};
