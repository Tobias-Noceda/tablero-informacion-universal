<script lang="ts">
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import { describeError, passwordError } from '$lib/auth/form';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';

	let password = $state('');
	let confirmation = $state('');
	let error = $state('');
	let busy = $state(false);

	const passwordProblem = $derived(password ? passwordError(password) : undefined);
	const mismatch = $derived(
		confirmation && confirmation !== password ? m['auth.error_passwords_differ']() : undefined
	);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (passwordProblem || mismatch) return;
		error = '';
		busy = true;
		try {
			await session.reset(page.url.searchParams.get('token') ?? '', password);
			await goto(resolve('/'), { replaceState: true });
		} catch (err) {
			error = describeError(err);
			busy = false;
		}
	}
</script>

<h1 class="text-xl font-semibold">{m['auth.reset_title']()}</h1>

<form class="flex flex-col gap-3" onsubmit={submit}>
	<Input
		id="password"
		name="password"
		type="password"
		autocomplete="new-password"
		label={m['auth.password']()}
		bind:value={password}
		errorMessage={passwordProblem}
		required
	/>
	<Input
		id="confirmation"
		name="confirmation"
		type="password"
		autocomplete="new-password"
		label={m['auth.password_confirm']()}
		bind:value={confirmation}
		errorMessage={mismatch}
		required
	/>

	{#if error}
		<p role="alert" class="rounded-md bg-error-bg p-3 text-sm text-error-text">{error}</p>
	{/if}

	<Button
		variant="primary"
		type="submit"
		disabled={busy || !password || password !== confirmation || !!passwordProblem}
	>
		{m['auth.reset_save']()}
	</Button>
</form>

<a href={resolve('/login')} class="text-sm underline hover:text-main">{m['auth.back_to_login']()}</a
>
