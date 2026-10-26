<script lang="ts">
	import Button from '$components/Button/Button.svelte';
	import Dropdown from '$components/Dropdown/Dropdown.svelte';
	import Input from '$components/Input/Input.svelte';
	import { m } from '$lib/paraglide/messages';
	import { getLocale } from '$lib/paraglide/runtime';
	import * as tokens from '$services/tokens';
	import type { AccessToken, CreatedAccessToken, UUID } from '$types/api';

	interface Props {
		user: UUID;
	}

	let { user }: Props = $props();

	let list = $state<AccessToken[]>([]);
	let name = $state('');
	let days = $state<tokens.Lifetime>('15');
	let creating = $state(false);
	let error = $state('');
	let created = $state<CreatedAccessToken | null>(null);
	let copied = $state('');
	let fields = $state<HTMLTextAreaElement[]>([]);

	$effect(() => {
		tokens.list(user).then((found) => (list = found));
	});

	// Same origin as the app: the MCP server is behind the same router.
	const mcpUrl = `${window.location.origin}/mcp`;

	const snippets = $derived(
		created
			? [
					`claude mcp add --transport http tiu ${mcpUrl} --header "Authorization: Bearer ${created.token}"`,
					JSON.stringify(
						{
							mcpServers: {
								tiu: { url: mcpUrl, headers: { Authorization: `Bearer ${created.token}` } }
							}
						},
						null,
						2
					)
				]
			: []
	);

	const date = (iso: string) =>
		new Date(iso).toLocaleDateString(getLocale(), { dateStyle: 'medium' });

	function message(code: string) {
		switch (code) {
			case 'too_many_tokens':
				return m['profile.tokens.too_many']();
			case 'rate_limited':
				return m['profile.tokens.rate_limited']();
			default:
				return m['profile.tokens.create_failed']();
		}
	}

	async function create(event: SubmitEvent) {
		event.preventDefault();
		creating = true;
		error = '';
		try {
			created = await tokens.create(user, name.trim(), days);
			list = [created, ...list];
			name = '';
		} catch (err) {
			console.error(err);
			error = message(err instanceof tokens.TokenError ? err.code : 'unknown');
		} finally {
			creating = false;
		}
	}

	async function revoke(token: AccessToken) {
		if (!confirm(m['profile.tokens.revoke_confirm']({ name: token.name }))) return;
		try {
			await tokens.revoke(user, token.id);
			list = list.filter((t) => t.id !== token.id);
		} catch (err) {
			console.error(err);
			error = m['profile.tokens.revoke_failed']();
		}
	}

	// navigator.clipboard only exists in a secure context; over plain HTTP on
	// a LAN address the text stays selected for a manual copy instead.
	async function copy(text: string, field: HTMLTextAreaElement | undefined) {
		field?.select();
		try {
			await navigator.clipboard.writeText(text);
			copied = text;
		} catch {
			// Left selected: Ctrl+C copies it.
		}
	}
</script>

<section class="flex flex-col gap-3">
	<h2 class="text-sm font-medium">{m['profile.tokens.title']()}</h2>
	<p class="text-sm opacity-70">{m['profile.tokens.hint']()}</p>

	{#if created}
		<div class="flex flex-col gap-3 rounded-md border border-main-border p-3">
			<h3 class="font-medium">{m['profile.tokens.created_title']()}</h3>
			<p class="text-sm opacity-70">{m['profile.tokens.created_hint']()}</p>
			{#each [created.token, ...snippets] as text, i (text)}
				<div class="flex items-start gap-2">
					<textarea
						bind:this={fields[i]}
						class="flex-1 resize-none rounded-md border border-border bg-transparent p-2 font-mono text-xs"
						rows={text.split('\n').length}
						readonly
						value={text}
						onfocus={(e) => e.currentTarget.select()}
					></textarea>
					<Button variant="tertiary" type="button" onclick={() => copy(text, fields[i])}>
						{copied === text ? m['profile.tokens.copied']() : m['profile.tokens.copy']()}
					</Button>
				</div>
			{/each}
			<Button variant="primary" type="button" onclick={() => (created = null)}>
				{m['profile.tokens.done']()}
			</Button>
		</div>
	{/if}

	<form class="flex flex-col gap-3" onsubmit={create}>
		<Input
			id="token-name"
			name="token-name"
			label={m['profile.tokens.name']()}
			placeholder={m['profile.tokens.name_placeholder']()}
			bind:value={name}
			required
			class="focus:ring-0 focus:border-sidebar!"
		/>
		<Dropdown
			label={m['profile.tokens.expiry']()}
			bind:value={days}
			options={tokens.LIFETIMES.map((lifetime) => ({
				value: lifetime,
				label: m['profile.tokens.days']({ days: lifetime })
			}))}
			class="w-full"
			activatorClass="rounded-md border border-border bg-transparent px-3 py-2 w-full!"
			drawerClass="rounded-md"
		/>
		<div class="flex items-center gap-3">
			<Button variant="primary" type="submit" disabled={creating || !name.trim()}>
				{m['profile.tokens.create']()}
			</Button>
			{#if error}
				<span role="alert" class="text-sm text-destructive">{error}</span>
			{/if}
		</div>
	</form>

	{#if list.length === 0}
		<p class="text-sm opacity-70">{m['profile.tokens.empty']()}</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each list as token (token.id)}
				<li
					class="flex items-center justify-between gap-3 rounded-md border border-border px-3 py-2"
				>
					<div class="flex flex-col text-sm">
						<span class="font-medium">{token.name}</span>
						<span class="font-mono text-xs opacity-70">{token.prefix}…</span>
						<span class="text-xs opacity-70">
							{m['profile.tokens.expires']({ date: date(token.expires_at) })} ·
							{token.last_used_at
								? m['profile.tokens.last_used']({ date: date(token.last_used_at) })
								: m['profile.tokens.never_used']()}
						</span>
					</div>
					<Button variant="destructive" type="button" onclick={() => revoke(token)}>
						{m['profile.tokens.revoke']()}
					</Button>
				</li>
			{/each}
		</ul>
	{/if}
</section>
