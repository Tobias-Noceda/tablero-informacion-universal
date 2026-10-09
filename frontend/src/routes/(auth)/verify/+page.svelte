<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import { describeError } from '$lib/auth/form';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';

	let error = $state('');

	// The mailed link is single use: redeem it once, then drop it from history.
	onMount(async () => {
		try {
			await session.verify(page.url.searchParams.get('token') ?? '');
			await goto(resolve('/'), { replaceState: true });
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
	<p class="opacity-70">{m['auth.verifying']()}</p>
{/if}
