import { error } from '@sveltejs/kit';
import * as boardApi from '$services/board';
import { orgs } from '$stores/org.svelte';
import { wellKnowns } from '$stores/well-knowns.svelte';
import type { UUID } from '$types/api';
import type { BoardData } from '$layouts/BoardContent/type';
import { nodeType } from '$lib/well-knowns/one-off';

// A board's page data, only under the organization it is filed under
// (undefined for Personal); any other combination is not found.
export async function loadBoard(id: UUID, orgId?: UUID): Promise<BoardData> {
	// The card types (catalogue and the board's custom ones) decide how each
	// card renders and what its form asks.
	const [board] = await Promise.all([
		boardApi.get(id),
		orgs.ensure(),
		wellKnowns.ensure(),
		wellKnowns.loadBoard(id)
	]);

	if (orgs.orgOf(board) !== orgId) error(404);
	orgs.remember(orgId);

	return {
		boardId: id,
		boardName: board.name,
		orgId,
		// A card without a type is a one-off card.
		nodes: board.postits.map((p) => ({ ...p, type: nodeType(p.type) })),
		edges: board.strands,
		role: board.role ?? 'viewer'
	};
}
