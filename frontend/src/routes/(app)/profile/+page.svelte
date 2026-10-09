<script lang="ts">
	import { resolve } from '$app/paths';
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';
	import * as users from '$services/users';
	import type { IdentityProvider } from '$types/api';

	// The (app) guard only renders this for a signed-in user.
	const user = $derived(session.user!);

	let name = $state(session.user?.name ?? '');
	let status = $state<'idle' | 'saving' | 'saved' | 'failed'>('idle');

	const identityLabel: Record<IdentityProvider, () => string> = {
		google: m['profile.identity_google'],
		password: m['profile.identity_password']
	};

	async function save(event: SubmitEvent) {
		event.preventDefault();
		status = 'saving';
		try {
			session.user = await users.rename(user.id, name.trim());
			status = 'saved';
		} catch (err) {
			console.error(err);
			status = 'failed';
		}
	}
</script>

<main class="mx-auto flex w-full max-w-lg flex-col gap-6 p-8">
	<h1 class="text-2xl font-semibold">{m['profile.title']()}</h1>

	<form class="flex flex-col gap-3" onsubmit={save}>
		<Input
			id="name"
			name="name"
			autocomplete="name"
			label={m['profile.name']()}
			bind:value={name}
			oninput={() => (status = 'idle')}
			required
		/>
		<div class="flex items-center gap-3">
			<Button
				variant="primary"
				type="submit"
				disabled={status === 'saving' || !name.trim() || name.trim() === user.name}
			>
				{m['profile.save']()}
			</Button>
			{#if status === 'saved'}
				<span class="text-sm text-success">{m['profile.saved']()}</span>
			{:else if status === 'failed'}
				<span role="alert" class="text-sm text-destructive">{m['profile.save_failed']()}</span>
			{/if}
		</div>
	</form>

	<section class="flex flex-col gap-1">
		<h2 class="text-sm font-medium">{m['profile.email']()}</h2>
		<p class="flex items-center gap-2">
			{user.email}
			<span
				class={[
					'rounded-full px-2 py-0.5 text-xs',
					user.email_verified ? 'bg-success text-white' : 'bg-skeleton'
				]}
			>
				{user.email_verified ? m['profile.verified']() : m['profile.unverified']()}
			</span>
		</p>
	</section>

	<section class="flex flex-col gap-2">
		<h2 class="text-sm font-medium">{m['profile.identities']()}</h2>
		<ul class="flex gap-2">
			{#each user.identities as identity (identity)}
				<li class="rounded-md border border-border px-3 py-1 text-sm">
					{identityLabel[identity]()}
				</li>
			{/each}
		</ul>
		{#if !user.identities.includes('password')}
			<p class="text-sm opacity-70">
				{m['profile.add_password_hint']()}
				<a class="underline" href={resolve(`/forgot?email=${encodeURIComponent(user.email)}`)}>
					{m['profile.add_password']()}
				</a>
			</p>
		{/if}
	</section>
</main>
