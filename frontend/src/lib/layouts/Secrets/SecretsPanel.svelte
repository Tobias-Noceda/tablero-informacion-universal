<script lang="ts">
	import Button from '$components/Button/Button.svelte';
	import Input from '$components/Input/Input.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import * as secretsApi from '$services/secrets';
	import { session } from '$modules/session.svelte';
	import type { OAuth2Flow, OAuthProvider, OAuthProviderStatus, SecretMeta, UUID } from '$types/api';

	type StaticKind = 'api_key' | 'bearer' | 'basic';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils';
	import { boardPlan, plan } from '$stores/plan.svelte';
	import { canAddOwnSecret, canAddSecret } from '$lib/billing/can';
	import PlanHint from '$components/PlanHint/PlanHint.svelte';

	let { board, onclose }: { board: UUID; onclose: () => void } = $props();

	// "board": shared with every member. "mine": what this user keeps here
	// for themselves; nobody else on the board can see or bind it.
	type Tab = 'board' | 'mine';
	let tab = $state<Tab>('board');
	const scope = $derived(
		tab === 'board' ? secretsApi.boardScope(board) : secretsApi.memberScope(board, session.userId)
	);

	// A board credential counts against whoever pays for the board, one of
	// "only mine" against the caller. With no room left there is no form.
	const canAdd = $derived(
		tab === 'board'
			? canAddSecret(boardPlan.secretsLeft)
			: canAddOwnSecret(plan.limits, plan.usage?.secrets)
	);
	const limitHint = $derived(
		tab === 'board'
			? { count: boardPlan.limits.secrets, max: boardPlan.limits.secrets }
			: { count: plan.usage?.secrets ?? plan.limits.secrets, max: plan.limits.secrets }
	);

	// What is left after a credential was added or removed.
	function recount() {
		void boardPlan.reload();
		void plan.refreshUsage();
	}
	recount();

	let secrets = $state<SecretMeta[]>([]);
	let providers = $state<OAuthProviderStatus[]>([]);
	let loading = $state(true);
	let error = $state('');

	type Draft = 'static' | 'oauth2' | 'connect';
	let drafting = $state<Draft | null>(null);
	let provider = $state<OAuthProvider>('google');

	let name = $state('');
	let value = $state('');
	let kind = $state<StaticKind>('api_key');
	let basicUser = $state('');
	let basicPassword = $state('');
	let flow = $state<OAuth2Flow>('client_credentials');
	let clientId = $state('');
	let clientSecret = $state('');
	let tokenUrl = $state('');
	let authUrl = $state('');
	let scopes = $state('');

	const nameLooksValid = $derived(/^[A-Z][A-Z0-9_]{0,63}$/.test(name));

	const staticValue = $derived(
		kind === 'basic' ? `${basicUser}:${basicPassword}` : value.trim()
	);

	const canSave = $derived(
		drafting === 'static'
			? nameLooksValid &&
					(kind === 'basic' ? basicUser !== '' && basicPassword !== '' : value.trim() !== '')
			: drafting === 'connect'
				? nameLooksValid
				: nameLooksValid &&
					clientId.trim() !== '' &&
					clientSecret.trim() !== '' &&
					tokenUrl.trim() !== '' &&
					(flow === 'client_credentials' || authUrl.trim() !== '')
	);

	// Only providers the platform has an application for can be offered.
	const connectable = $derived(providers.filter((p) => p.configured));

	async function refresh() {
		loading = true;
		error = '';
		try {
			[secrets, providers] = await Promise.all([secretsApi.list(scope), secretsApi.providers()]);
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
		kind = 'api_key';
		basicUser = '';
		basicPassword = '';
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
				await secretsApi.put(scope, name, kind, staticValue);
			} else if (drafting === 'connect') {
				// The grant is created server-side and the user is sent to consent;
				// the panel is left behind, so there is nothing to refresh here.
				const redirect = `${window.location.origin}/oauth2/callback`;
				window.location.href = await secretsApi.connect(scope, provider, name, redirect);
				return;
			} else {
				await secretsApi.put_oauth2(scope, {
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
		recount();
	}

	async function remove(secret: SecretMeta) {
		error = '';
		try {
			await secretsApi.del(scope, secret.name);
			await refresh();
			recount();
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	async function consent(secret: SecretMeta) {
		error = '';
		try {
			const redirect = `${window.location.origin}/oauth2/callback`;
			const target = await secretsApi.authorize(scope, secret.name, redirect);
			window.location.href = target;
		} catch (e) {
			error = e instanceof Error ? e.message : String(e);
		}
	}

	// A credential being written stays: it is saved in whichever tab is shown.
	function show(next: Tab) {
		tab = next;
	}

	// Re-runs when the tab changes, since the scope is derived from it.
	$effect(() => {
		void scope;
		refresh();
	});
</script>

<Modal onclose={onclose} onaccept={onclose} acceptText={m['secrets.close']()}>
	<h2 class="text-lg font-semibold">{m['secrets.title']()}</h2>

	<div class="flex gap-1 border-b border-main-border" role="tablist">
		{#each [['board', m['secrets.tab_board']()], ['mine', m['secrets.tab_mine']()]] as [key, label] (key)}
			<button
				type="button"
				role="tab"
				aria-selected={tab === key}
				class={cn(
					'px-3 py-1 text-sm border-b-2 -mb-px',
					tab === key ? 'border-primary font-semibold' : 'border-transparent opacity-70'
				)}
				onclick={() => show(key as Tab)}
			>
				{label}
			</button>
		{/each}
	</div>
	{#if tab === 'mine'}
		<p class="text-xs opacity-70">{m['secrets.mine_hint']()}</p>
	{/if}

	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{/if}

	{#if loading}
		<p class="text-sm opacity-70">{m['secrets.loading']()}</p>
	{:else if secrets.length === 0}
		<p class="text-sm opacity-70">{tab === 'mine' ? m['secrets.empty_mine']() : m['secrets.empty']()}</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each secrets as secret (secret.name)}
				<li class="flex items-center justify-between gap-2 border border-main-border rounded-md px-3 py-2">
					<div class="flex flex-col">
						<code class="text-sm">${secret.name}</code>
						<span class="text-xs opacity-60">
							{secret.provider ?? secret.kind}{secret.flow && !secret.provider ? ` · ${secret.flow}` : ''}
							{#if secret.flow === 'authorization_code'}
								· {secret.authorized ? m['secrets.authorized']() : m['secrets.pending']()}
							{/if}
						</span>
					</div>
					<div class="flex gap-2">
						<!-- Only the code flow has a user to send to a consent screen;
						     client_credentials needs no handshake at all. -->
						{#if secret.flow === 'authorization_code'}
							<Button variant="secondary" class="text-xs px-2 py-1" onclick={() => consent(secret)}>
								{secret.authorized ? m['secrets.reauthorize']() : m['secrets.authorize']()}
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

	{#if !canAdd}
		<PlanHint text={m['billing.hint.secrets'](limitHint)} />
	{:else if drafting === null}
		<div class="flex gap-2">
			<Button variant="primary" onclick={() => (drafting = 'static')}>
				{m['secrets.add_static']()}
			</Button>
			<Button variant="secondary" onclick={() => (drafting = 'oauth2')}>
				{m['secrets.add_oauth2']()}
			</Button>
			{#if connectable.length > 0}
				<Button variant="secondary" onclick={() => (drafting = 'connect')}>
					{m['secrets.connect_account']()}
				</Button>
			{/if}
		</div>
	{:else}
		<div class="flex flex-col gap-2 border-t border-main-border pt-3">
			<Input label={m['secrets.name']()} placeholder="API_KEY" bind:value={name} required />
			{#if name !== '' && !nameLooksValid}
				<p class="text-xs text-destructive">{m['secrets.name_hint']()}</p>
			{/if}

			{#if drafting === 'static'}
				<Dropdown
					floating
					variant="plain"
					label={m['secrets.method']()}
					class="w-full"
					options={(['api_key', 'bearer', 'basic'] as const).map((k) => ({
						value: k,
						label: m[`secrets.method_${k}`]()
					}))}
					value={kind}
					onchange={(selected) => (kind = selected as StaticKind)}
				/>
				<p class="text-xs opacity-70">{m[`secrets.method_${kind}_hint`]()}</p>

				{#if kind === 'basic'}
					<Input label={m['secrets.username']()} bind:value={basicUser} required />
					<Input label={m['secrets.password']()} type="password" bind:value={basicPassword} required />
				{:else}
					<Input label={m['secrets.value']()} type="password" bind:value required />
				{/if}
			{:else if drafting === 'connect'}
				<Dropdown
					floating
					variant="plain"
					label={m['secrets.provider']()}
					class="w-full"
					options={connectable.map((p) => ({ value: p.provider, label: p.provider }))}
					value={provider}
					onchange={(selected) => (provider = selected as OAuthProvider)}
				/>
				<p class="text-xs opacity-70">{m['secrets.connect_hint']()}</p>
			{:else}
				<Dropdown
					floating
					variant="plain"
					label={m['secrets.flow']()}
					class="w-full"
					options={(['client_credentials', 'authorization_code'] as const).map((f) => ({
						value: f,
						label: f
					}))}
					value={flow}
					onchange={(selected) => (flow = selected as OAuth2Flow)}
				/>
				<Input label={m['secrets.client_id']()} bind:value={clientId} required />
				<Input label={m['secrets.client_secret']()} type="password" bind:value={clientSecret} required />
				<Input label={m['secrets.token_url']()} placeholder="https://provider.example/oauth/token" bind:value={tokenUrl} required />
				{#if flow === 'authorization_code'}
					<Input label={m['secrets.auth_url']()} placeholder="https://provider.example/oauth/authorize" bind:value={authUrl} required />
				{/if}
				<Input label={m['secrets.scopes']()} placeholder="read write" bind:value={scopes} />
			{/if}

			{#if drafting !== 'connect'}
				<p class="text-xs opacity-70">{m['secrets.write_only']()}</p>
			{/if}

			<div class="flex gap-2">
				<Button variant="primary" disabled={!canSave} onclick={save}>
					{drafting === 'connect' ? m['secrets.connect']() : m['secrets.save']()}
				</Button>
				<Button variant="gray" onclick={resetDraft}>{m['secrets.cancel']()}</Button>
			</div>
		</div>
	{/if}
</Modal>
