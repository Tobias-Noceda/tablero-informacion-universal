import { session } from '$modules/session.svelte';
import type { PageLoad } from './$types';

// The plans are for anyone: a visitor sees them signed out, a user sees
// their own plan marked. Nobody is sent to sign in from here.
export const load: PageLoad = async () => {
	await session.bootstrap();
};
