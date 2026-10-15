import type { PageLoad } from './$types';
import * as orgsApi from '$services/orgs';

export const load = (async ({ params, parent, depends }) => {
	await parent();
	depends('app:org');

	return { org: await orgsApi.get(params.id) };
}) satisfies PageLoad;
