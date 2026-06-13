import type { PageLoad } from './$types';
import * as boardApi from '$services/board';

export const load = (async ({ params }) => {
    const id = params.id;

    const { postits, strands, name } = await boardApi.get(id);

    return { 
        nodes: postits,
        edges: strands,
        name
    };
}) satisfies PageLoad;