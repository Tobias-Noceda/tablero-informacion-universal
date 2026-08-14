<script lang="ts">
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import * as secretsApi from '$services/secrets';
	import type { OAuth2Flow, SecretMeta, UUID } from '$types/api';
	import { m } from '$lib/paraglide/messages';

	let { board, onclose }: { board: UUID; onclose: () => void } = $props();

	let secrets = $state<SecretMeta[]>([]);
	let loading = $state(true);
	let error = $state('');

	type Draft = 'static' | 'oauth2';
	let drafting = $state<Draft | null>(null);

	let name = $state('');
	let value = $state('');
	let flow = $state<OAuth2Flow>('client_credentials');
	let clientId = $state('');
	let clientSecret = $state('');
	let tokenUrl = $state('');
	let authUrl = $state('');
	let scopes = $state('');

	const nameLooksValid = $derived(/^[A-Z][A-Z0-9_]{0,63}$/.test(name));

	const canSave = $derived(
		drafting === 'static'
			? nameLooksValid && value.trim() !== ''
			: nameLooksValid &&
					clientId.trim() !== '' &&
					clientSecret.trim() !== '' &&
					tokenUrl.trim() !== '' &&
					(flow === 'client_credentials' || authUrl.trim() !== '')
	);

	async function refresh() {
		loading = true;
		error = '';
		try {
			secrets = await secretsApi.list(board);
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		} finally {
			loading = false;
		}
	}

	function resetDraft() {
		drafting = null;
		name = '';
		value = '';
		clientId = '';
		clientSecret = '';
		tokenUrl = '';
		authUrl = '';
		scopes = '';
		flow = 'client_credentials';
	}

	async function save() {
		error = '';
		try {
			if (drafting === 'static') {
				await secretsApi.put(board, name, 'api_key', value);
			} else {
				await secretsApi.put_oauth2(board, {
					name,
					flow,
					client_id: clientId.trim(),
					client_secret: clientSecret,
					token_url: tokenUrl.trim(),
					auth_url: flow === 'authorization_code' ? authUrl.trim() : undefined,
					scopes: scopes.trim() || undefined
				});
			}
			resetDraft();
			await refresh();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	async function remove(secret: SecretMeta) {
		error = '';
		try {
			await secretsApi.del(board, secret.name);
			await refresh();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	async function consent(secret: SecretMeta) {
		error = '';
		try {
			const redirect = `${window.location.origin}/oauth2/callback`;
			const target = await secretsApi.authorize(board, secret.name, redirect);
			window.location.href = target;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	$effect(() => {
		refresh();
	});
</script>

<Modal onclose={onclose} onaccept={onclose} acceptText={m['secrets.close']()}>
	<h2 class="text-lg font-semibold">{m['secrets.title']()}</h2>

	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{/if}

	{#if loading}
		<p class="text-sm opacity-70">{m['secrets.loading']()}</p>
	{:else if secrets.length === 0}
		<p class="text-sm opacity-70">{m['secrets.empty']()}</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each secrets as secret (secret.name)}
				<li class="flex items-center justify-between gap-2 border border-main-border rounded-md px-3 py-2">
					<div class="flex flex-col">
						<code class="text-sm">${secret.name}</code>
						<span class="text-xs opacity-60">{secret.kind}</span>
					</div>
					<div class="flex gap-2">
						{#if secret.kind === 'oauth2'}
							<Button variant="secondary" class="text-xs px-2 py-1" onclick={() => consent(secret)}>
								{m['secrets.authorize']()}
							</Button>
						{/if}
						<Button variant="destructive" class="text-xs px-2 py-1" onclick={() => remove(secret)}>
							{m['secrets.delete']()}
						</Button>
					</div>
				</li>
			{/each}
		</ul>
	{/if}

	{#if drafting === null}
		<div class="flex gap-2">
			<Button variant="primary" onclick={() => (drafting = 'static')}>
				{m['secrets.add_static']()}
			</Button>
			<Button variant="secondary" onclick={() => (drafting = 'oauth2')}>
				{m['secrets.add_oauth2']()}
			</Button>
		</div>
	{:else}
		<div class="flex flex-col gap-2 border-t border-main-border pt-3">
			<Input label={m['secrets.name']()} placeholder="API_KEY" bind:value={name} required />
			{#if name !== '' && !nameLooksValid}
				<p class="text-xs text-destructive">{m['secrets.name_hint']()}</p>
			{/if}

			{#if drafting === 'static'}
				<Input label={m['secrets.value']()} type="password" bind:value required />
			{:else}
				<label class="flex flex-col gap-1 text-sm">
					{m['secrets.flow']()}
					<select class="bg-background border border-main-border rounded-md px-2 py-1" bind:value={flow}>
						<option value="client_credentials">client_credentials</option>
						<option value="authorization_code">authorization_code</option>
					</select>
				</label>
				<Input label={m['secrets.client_id']()} bind:value={clientId} required />
				<Input label={m['secrets.client_secret']()} type="password" bind:value={clientSecret} required />
				<Input label={m['secrets.token_url']()} placeholder="https://provider.example/oauth/token" bind:value={tokenUrl} required />
				{#if flow === 'authorization_code'}
					<Input label={m['secrets.auth_url']()} placeholder="https://provider.example/oauth/authorize" bind:value={authUrl} required />
				{/if}
				<Input label={m['secrets.scopes']()} placeholder="read write" bind:value={scopes} />
			{/if}

			<p class="text-xs opacity-70">{m['secrets.write_only']()}</p>

			<div class="flex gap-2">
				<Button variant="primary" disabled={!canSave} onclick={save}>{m['secrets.save']()}</Button>
				<Button variant="gray" onclick={resetDraft}>{m['secrets.cancel']()}</Button>
			</div>
		</div>
	{/if}
</Modal>
