<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { describeError, safeNext } from '$lib/auth/form';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';

	let error = $state('');

	// Google sends the browser here with ?code=&state= (or ?error= when the
	// user backs out). Only the backend can redeem them: it holds the PKCE
	// verifier, and the tiu_signin cookie proves this browser started it.
	onMount(async () => {
		const params = page.url.searchParams;
		if (params.has('error')) {
			error = m['auth.error_google_denied']();
			return;
		}

		try {
			const next = await session.googleCallback(
				params.get('code') ?? '',
				params.get('state') ?? ''
			);
			await goto(safeNext(next), { replaceState: true });
		} catch (err) {
			error = describeError(err);
		}
	});
</script>

{#if error}
	<p role="alert" class="rounded-md bg-error-bg p-3 text-sm text-error-text">{error}</p>
	<a href={resolve('/login')} class="text-sm underline hover:text-main"
		>{m['auth.back_to_login']()}</a
	>
{:else}
	<p class="opacity-70">{m['auth.google_signing_in']()}</p>
{/if}
