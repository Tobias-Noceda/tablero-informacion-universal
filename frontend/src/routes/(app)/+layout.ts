import { redirect } from '@sveltejs/kit';
import { session } from '$modules/session.svelte';
import type { LayoutLoad } from './$types';

// Everything in (app) needs a user: the first visit trades the refresh cookie
// for a session, anyone without one goes to sign in and comes back here. The
// bare home address shows what the app is about instead.
export const load: LayoutLoad = async ({ url }) => {
	await session.bootstrap();

	if (session.status !== 'authenticated') {
		if (url.pathname === '/') redirect(302, '/about');
		redirect(302, `/login?${new URLSearchParams({ next: url.pathname + url.search })}`);
	}
};
