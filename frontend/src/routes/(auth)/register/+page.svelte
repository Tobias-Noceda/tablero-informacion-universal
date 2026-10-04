<script lang="ts">
	import { resolve } from '$app/paths';
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import { describeError, passwordError } from '$lib/auth/form';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';

	let name = $state('');
	let email = $state('');
	let password = $state('');
	let error = $state('');
	let busy = $state(false);
	let sentTo = $state('');

	const passwordProblem = $derived(password ? passwordError(password) : undefined);

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		if (passwordProblem) return;
		error = '';
		busy = true;
		try {
			await session.register(email, password, name);
			sentTo = email;
		} catch (err) {
			error = describeError(err);
		} finally {
			busy = false;
		}
	}
</script>

{#if sentTo}
	<h1 class="text-xl font-semibold">{m['auth.check_inbox_title']()}</h1>
	<p>{m['auth.check_inbox']({ email: sentTo })}</p>
	<a href={resolve('/login')} class="text-sm underline hover:text-main"
		>{m['auth.back_to_login']()}</a
	>
{:else}
	<h1 class="text-xl font-semibold">{m['auth.register_title']()}</h1>

	<form class="flex flex-col gap-3" onsubmit={submit}>
		<Input
			id="name"
			name="name"
			autocomplete="name"
			label={m['auth.name']()}
			bind:value={name}
			required
		/>
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
			autocomplete="new-password"
			label={m['auth.password']()}
			bind:value={password}
			errorMessage={passwordProblem}
			required
		/>

		{#if error}
			<p role="alert" class="rounded-md bg-error-bg p-3 text-sm text-error-text">{error}</p>
		{/if}

		<Button
			variant="primary"
			type="submit"
			disabled={busy || !name.trim() || !email || !password || !!passwordProblem}
		>
			{m['auth.register']()}
		</Button>
	</form>

	<p class="text-center text-sm">
		{m['auth.have_account']()}
		<a href={resolve('/login')} class="underline hover:text-main">{m['auth.login_link']()}</a>
	</p>
{/if}
