import type { PageLoad } from './$types';
import * as boardApi from '$services/board';

export const load = (async ({ params, parent }) => {
	await parent();
	const id = params.id;

	const { postits, strands, name, role } = await boardApi.get(id);

	return {
		boardId: id,
		boardName: name,
		nodes: postits,
		edges: strands,
		role: role ?? 'viewer'
	};
}) satisfies PageLoad;
