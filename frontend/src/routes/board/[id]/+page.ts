import type { PageLoad } from './$types';
import * as boardApi from '$services/board';
import { uuid } from '$lib/utils';

export const load = (async ({ params }) => {
	const id = params.id;

	const { postits, strands, name } = await boardApi.get(id);

	return {
		userId: uuid(),
		nodes: postits,
		edges: strands,
		name
	};
}) satisfies PageLoad;
