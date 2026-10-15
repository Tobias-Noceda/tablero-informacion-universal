<script lang="ts">
	import { page } from '$app/state';
	import { resolve } from '$app/paths';
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import { describeError } from '$lib/auth/form';
	import { m } from '$lib/paraglide/messages';
	import { session } from '$modules/session.svelte';

	// The profile sends a Google-only user here with their address filled in.
	let email = $state(page.url.searchParams.get('email') ?? '');
	let error = $state('');
	let busy = $state(false);
	let sentTo = $state('');

	async function submit(event: SubmitEvent) {
		event.preventDefault();
		error = '';
		busy = true;
		try {
			await session.forgot(email);
			sentTo = email;
		} catch (err) {
			error = describeError(err);
		} finally {
			busy = false;
		}
	}
</script>

<h1 class="text-xl font-semibold">{m['auth.forgot_title']()}</h1>

{#if sentTo}
	<p>{m['auth.forgot_sent']({ email: sentTo })}</p>
{:else}
	<p class="text-sm opacity-80">{m['auth.forgot_hint']()}</p>

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

		{#if error}
			<p role="alert" class="rounded-md bg-error-bg p-3 text-sm text-error-text">{error}</p>
		{/if}

		<Button variant="primary" type="submit" disabled={busy || !email}
			>{m['auth.forgot_send']()}</Button
		>
	</form>
{/if}

<a href={resolve('/login')} class="text-sm underline hover:text-main">{m['auth.back_to_login']()}</a
>
