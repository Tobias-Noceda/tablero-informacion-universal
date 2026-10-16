import { redirect } from '@sveltejs/kit';
import { safeNext } from '$lib/auth/form';
import { session } from '$modules/session.svelte';
import type { PageLoad } from './$types';

export const load: PageLoad = async ({ parent, url }) => {
	await parent();
	if (session.status === 'authenticated') redirect(302, safeNext(url.searchParams.get('next')));
};
