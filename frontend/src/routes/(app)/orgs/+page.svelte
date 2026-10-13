<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import { m } from '$lib/paraglide/messages';
	import * as orgsApi from '$services/orgs';
	import { orgs } from '$stores/org.svelte';

	let name = $state('');
	let creating = $state(false);
	let failed = $state(false);

	onMount(() => {
		orgs.refresh().catch((err) => console.error('Failed to load organizations', err));
	});

	async function create(event: SubmitEvent) {
		event.preventDefault();
		creating = true;
		failed = false;
		try {
			const org = await orgsApi.create(name.trim());
			await orgs.refresh();
			orgs.select(org.id);
			await goto(resolve(`/orgs/${org.id}`));
		} catch (err) {
			console.error(err);
			failed = true;
		} finally {
			creating = false;
		}
	}
</script>

<main class="mx-auto flex w-full max-w-lg flex-col gap-6 p-8">
	<header class="flex flex-col gap-1">
		<h1 class="text-2xl font-semibold">{m['orgs.title']()}</h1>
		<p class="text-sm opacity-70">{m['orgs.hint']()}</p>
	</header>

	{#if orgs.list.length === 0}
		<p class="text-sm opacity-70">{m['orgs.none']()}</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each orgs.list as org (org.id)}
				<li>
					<a
						href={resolve(`/orgs/${org.id}`)}
						class="flex items-center justify-between rounded-md border border-main-border px-3 py-2 hover:bg-skeleton"
					>
						<span class="truncate">{org.name}</span>
						<span class="text-sm opacity-70">{m[`orgs.role_${org.role}`]()}</span>
					</a>
				</li>
			{/each}
		</ul>
	{/if}

	<form class="flex flex-col gap-2 border-t border-main-border pt-4" onsubmit={create}>
		<Input id="org-name" label={m['orgs.name']()} bind:value={name} required />
		{#if failed}
			<p role="alert" class="text-sm text-destructive">{m['orgs.error_unknown']()}</p>
		{/if}
		<Button variant="primary" type="submit" disabled={creating || !name.trim()}>
			{m['orgs.create']()}
		</Button>
	</form>
</main>
