import { refreshBoards } from '$stores/boards';
import { orgs } from '$stores/org.svelte';
import type { PageLoad } from './$types';

// The Personal landing.
export const load = (async ({ parent }) => {
	await parent();
	orgs.remember(undefined);
	// Back from a board: its cards may have moved. The grid shows the old list meanwhile.
	refreshBoards().catch((err) => console.error('Failed to load boards', err));
}) satisfies PageLoad;
