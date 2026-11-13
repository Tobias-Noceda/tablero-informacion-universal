import type { BoardRole, CardStyle, UUID } from '$lib/types/api';
import type { StrandEdge } from '$lib/strands/style';

export interface BoardData {
    boardId: UUID;
    boardName: string;
    orgId?: UUID;
    nodes: Array<{
        id: UUID;
        type?: string;
        position: { x: number; y: number; };
    } & CardStyle>;
    edges: StrandEdge[];
    role: BoardRole;
}