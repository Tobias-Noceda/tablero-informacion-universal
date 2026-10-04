<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import { describeError, safeNext } from '$lib/auth/form';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';

	const next = $derived(safeNext(page.url.searchParams.get('next')));

	let email = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);

	// Covers this form and a sign-in finished in another tab alike.
	$effect(() => {
		if (session.status === 'authenticated') void goto(next, { replaceState: true });
	});

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		try {
			await session.login(email, password);
		} catch (err) {
			error = describeError(err);
		} finally {
			busy = false;
		}
	}

	async function continueWithGoogle() {
		error = '';
		busy = true;
		try {
			window.location.assign(await session.googleStart(next));
		} catch (err) {
			error = describeError(err);
			busy = false;
		}
	}
</script>

<h1 class="text-xl font-semibold">{m['auth.login_title']()}</h1>

<form class="flex flex-col gap-3" onsubmit={submit}>
	<Input
		id="email"
		name="email"
		type="email"
		autocomplete="email"
		label={m['auth.email']()}
		bind:value={email}
		required
	/>
	<Input
		id="password"
		name="password"
		type="password"
		autocomplete="current-password"
		label={m['auth.password']()}
		bind:value={password}
		required
	/>
	<a href={resolve('/forgot')} class="self-end text-sm underline hover:text-main">
		{m['auth.forgot_link']()}
	</a>

	{#if error}
		<p role="alert" class="rounded-md bg-error-bg p-3 text-sm text-error-text">{error}</p>
	{/if}

	<Button variant="primary" type="submit" disabled={busy || !email || !password}>
		{m['auth.login']()}
	</Button>
</form>

<div class="flex items-center gap-2 text-sm opacity-60">
	<span class="h-px flex-1 bg-border"></span>
	{m['auth.or']()}
	<span class="h-px flex-1 bg-border"></span>
</div>

<Button variant="tertiary" type="button" disabled={busy} onclick={continueWithGoogle}>
	{m['auth.google']()}
</Button>

<p class="text-center text-sm">
	{m['auth.no_account']()}
	<a href={resolve('/register')} class="underline hover:text-main">{m['auth.register_link']()}</a>
</p>
