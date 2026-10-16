import { redirect } from '@sveltejs/kit';
import { session } from '$modules/session.svelte';
import type { LayoutLoad } from './$types';

// Everything in (app) needs a user: the first visit trades the refresh cookie
// for a session, anyone without one goes to sign in and comes back here.
export const load: LayoutLoad = async ({ url }) => {
	await session.bootstrap();

	if (session.status !== 'authenticated') {
		redirect(302, `/login?${new URLSearchParams({ next: url.pathname + url.search })}`);
	}
};
