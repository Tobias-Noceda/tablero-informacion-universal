<script lang="ts">
	import { page } from '$app/state';
	import { m } from '$lib/paraglide/messages';
	import * as api from '$modules/api.svelte';

	// The provider redirects the browser here with ?state=&code=. Both are
	// handed straight to the backend, which is the only side that holds the
	// PKCE verifier and the client secret needed to redeem them.
	const params = $derived(page.url.searchParams);

	let done = $state(false);
	let error = $state('');

	$effect(() => {
		const denied = params.get('error');
		if (denied) {
			error = denied;
			return;
		}

		const query = new URLSearchParams({
			state: params.get('state') ?? '',
			code: params.get('code') ?? ''
		});

		api.get(`/v1/oauth2/callback?${query}`)
			.then(() => {
				done = true;
			})
			.catch((e) => {
				error = e instanceof Error ? e.message : String(e);
			});
	});
</script>

<main class="flex flex-col items-center justify-center h-full gap-2 p-8">
	{#if done}
		<p>{m['secrets.callback_ok']()}</p>
	{:else if error}
		<p class="text-destructive">{m['secrets.callback_fail']()}</p>
		<code class="text-xs opacity-70">{error}</code>
	{:else}
		<p class="opacity-70">{m['secrets.loading']()}</p>
	{/if}
</main>
