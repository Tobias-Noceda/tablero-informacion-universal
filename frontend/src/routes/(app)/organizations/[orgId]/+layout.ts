import { error } from '@sveltejs/kit';
import { refreshBoards } from '$stores/boards';
import { orgs } from '$stores/org.svelte';
import type { LayoutLoad } from './$types';

// An organization's landing, only for its members.
export const load = (async ({ params, parent }) => {
	await parent();
	await orgs.ensure();

	const org = orgs.byId(params.orgId);
	if (!org) error(404);
	orgs.remember(org.id);
	// Back from a board: its cards may have moved. The grid shows the old list meanwhile.
	refreshBoards().catch((err) => console.error('Failed to load boards', err));

	return { org };
}) satisfies LayoutLoad;
