import type { BoardRole, UUID } from '$lib/types/api';

export interface BoardData {
    boardId: UUID;
    boardName: string;
    orgId?: UUID;
    nodes: Array<{
        id: UUID;
        type?: string;
        position: { x: number; y: number; };
    }>;
    edges: Array<{
        source: UUID;
        target: UUID;
    }>;
    role: BoardRole;
}