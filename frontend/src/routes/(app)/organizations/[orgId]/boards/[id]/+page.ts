import type { PageLoad } from './$types';
import { loadBoard } from '$lib/helpers/board-load';

// A board of orgId: one filed elsewhere is not found here.
export const load = (async ({ params, parent }) => {
	await parent();
	return loadBoard(params.id, params.orgId);
}) satisfies PageLoad;
