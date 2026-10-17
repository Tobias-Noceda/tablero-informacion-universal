import type { LayoutLoad } from './$types';
import { loadBoard } from '$lib/helpers/board-load';

// A personal board: one filed under an organization is not found here.
export const load = (async ({ params, parent }) => {
	await parent();
	return loadBoard(params.id);
}) satisfies LayoutLoad;
