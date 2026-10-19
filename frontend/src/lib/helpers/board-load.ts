import { error } from '@sveltejs/kit';
import * as boardApi from '$services/board';
import { orgs } from '$stores/org.svelte';
import type { UUID } from '$types/api';
import type { BoardData } from '$layouts/BoardContent/type';

// A board's page data, only under the organization it is filed under
// (undefined for Personal); any other combination is not found.
export async function loadBoard(id: UUID, orgId?: UUID): Promise<BoardData> {
	const [board] = await Promise.all([boardApi.get(id), orgs.ensure()]);

	if (orgs.orgOf(board) !== orgId) error(404);
	orgs.remember(orgId);

	return {
		boardId: id,
		boardName: board.name,
		orgId,
		nodes: board.postits,
		edges: board.strands,
		role: board.role ?? 'viewer'
	};
}
