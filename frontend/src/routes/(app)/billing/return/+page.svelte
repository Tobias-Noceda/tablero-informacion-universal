<script lang="ts">
	import { onMount } from 'svelte';
	import { resolve } from '$app/paths';
	import { planName } from '$lib/billing/format';
	import { m } from '$lib/paraglide/messages';
	import * as billing from '$services/billing';
	import { plan } from '$stores/plan.svelte';
	import type { Subscription } from '$types/api';

	// Where Mercado Pago sends the user back. Its notification may not have
	// arrived yet, so the backend is asked to look the subscription up now.
	let outcome = $state<'checking' | 'active' | 'pending' | 'failed'>('checking');
	let subscription = $state<Subscription | null>(null);

	onMount(async () => {
		try {
			subscription = await billing.sync();
			outcome =
				subscription.status === 'authorized'
					? 'active'
					: subscription.status === 'pending'
						? 'pending'
						: 'failed';
		} catch (err) {
			console.error(err);
			outcome = 'failed';
		}
		// The plan and its limits travel with the profile.
		await plan.reloadProfile().catch(() => {});
	});
</script>

<main class="mx-auto flex w-full max-w-lg flex-col gap-4 p-8">
	<h1 class="text-2xl font-semibold">{m['billing.return.title']()}</h1>

	{#if outcome === 'checking'}
		<p class="text-sm opacity-70">{m['billing.return.checking']()}</p>
	{:else if outcome === 'active' && subscription}
		<p class="text-success">{m['billing.return.active']({ plan: planName(subscription.plan) })}</p>
	{:else if outcome === 'pending'}
		<p>{m['billing.return.pending']()}</p>
	{:else}
		<p role="alert" class="text-destructive">{m['billing.return.failed']()}</p>
	{/if}

	{#if outcome !== 'checking'}
		<div class="flex flex-wrap gap-4 text-sm">
			<a href={resolve('/')} class="underline hover:text-main">{m['billing.return.home']()}</a>
			{#if outcome === 'failed'}
				<a href={resolve('/pricing')} class="underline hover:text-main">
					{m['billing.return.retry']()}
				</a>
			{/if}
		</div>
	{/if}
</main>
