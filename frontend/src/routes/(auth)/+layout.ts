import { session } from '$modules/session.svelte';
import type { LayoutLoad } from './$types';

// Pages here work signed out, but must know whether someone is signed in.
export const load: LayoutLoad = async () => {
	await session.bootstrap();
};
