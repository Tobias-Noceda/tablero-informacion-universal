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

export type Strand = {
    id: UUID;
    source: UUID;
    target: UUID;
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

// What a card's query produced: GET /post-its/:id.
export type PostItData = Record<string, unknown>;
