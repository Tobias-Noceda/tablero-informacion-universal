<script lang="ts">
	import Button from '$components/Button/Button.svelte';
	import Modal from '$components/Modal/Modal.svelte';
	import { m } from '$lib/paraglide/messages';
	import { cn } from '$lib/utils';
	import { session } from '$modules/session.svelte';
	import * as templatesApi from '$services/postit-templates';
	import { orgs } from '$stores/org.svelte';
	import { wellKnowns } from '$stores/well-knowns.svelte';
	import { draftFrom, emptyDraft, type Draft } from '$lib/well-knowns/draft';
	import type { BoardRole, PostitTemplate, PostitTemplateBody, UUID } from '$types/api';
	import PostitTemplateForm from './PostitTemplateForm.svelte';

	interface Props {
		boardId: UUID;
		/** The organization the board is filed under, if any. */
		orgId?: UUID;
		role: BoardRole;
		onclose: () => void;
	}

	let { boardId, orgId, role, onclose }: Props = $props();

	const scrollClass = cn(
		// Custom scrollbar styles
		'scrollbar-thin scrollbar-thumb-main scrollbar-track-transparent',
		'[&::-webkit-scrollbar]:w-2',
		'[&::-webkit-scrollbar-track]:bg-transparent',
		'[&::-webkit-scrollbar-thumb]:bg-main',
		'[&::-webkit-scrollbar-thumb]:rounded-full',
		'[&::-webkit-scrollbar-thumb]:hover:bg-main-hover'
	);

	// "user": only the caller, on any board. "board": everyone on this board.
	// "org": every member of an organization, on its boards. Any organization
	// the user is in can be picked, not only the board's.
	type Tab = 'user' | 'board' | 'org';
	let tab = $state<Tab>('user');
	const tabs = $derived<Tab[]>(orgs.list.length > 0 ? ['user', 'board', 'org'] : ['user', 'board']);

	// The board's organization first, else one the user administers.
	// svelte-ignore state_referenced_locally
	let org = $state<UUID>(
		orgs.byId(orgId)?.id ?? (orgs.list.find((o) => o.role === 'admin') ?? orgs.list[0])?.id ?? ''
	);

	const scope = $derived(
		tab === 'user'
			? templatesApi.userScope(session.userId ?? '')
			: tab === 'board'
				? templatesApi.boardScope(boardId)
				: templatesApi.orgScope(org)
	);
	// Who may change what the tab lists; the backend checks it again.
	const canManage = $derived(
		tab === 'user' ||
			(tab === 'board' && (role === 'owner' || role === 'editor')) ||
			(tab === 'org' && orgs.byId(org)?.role === 'admin')
	);

	let list = $state<PostitTemplate[]>([]);
	let loading = $state(true);
	let error = $state('');
	// null: nothing open; 'new': a new one; otherwise the one being edited.
	let editing = $state<PostitTemplate | 'new' | null>(null);
	// A new card type being written. Kept across tabs: it is created in
	// whichever scope is shown when it is saved.
	let newDraft = $state<Draft>(emptyDraft());
	let editDraft = $state<Draft>(emptyDraft());

	function startNew() {
		editing = 'new';
	}

	function startEdit(template: PostitTemplate) {
		editDraft = draftFrom(template);
		editing = template;
	}

	function close(created: boolean) {
		if (editing === 'new' && created) newDraft = emptyDraft();
		editing = null;
	}

	function explain(e: unknown) {
		if (e instanceof templatesApi.PostitTemplateError) {
			if (e.code === 'limit_reached') return m['postit_templates.error_limit']();
			if (e.code === 'invalid_postit_template')
				return m['postit_templates.error_invalid']({ detail: e.detail ?? '' });
		}
		return m['postit_templates.error_generic']();
	}

	async function refresh() {
		// A tab switched while loading drops the previous tab's answer.
		const asked = tab;
		loading = true;
		error = '';
		try {
			const found = await templatesApi.list(scope);
			if (asked === tab) list = found;
		} catch (e) {
			if (asked !== tab) return;
			list = [];
			error = explain(e);
		} finally {
			if (asked === tab) loading = false;
		}
	}

	async function save(body: PostitTemplateBody) {
		error = '';
		try {
			if (editing === 'new') await templatesApi.create(scope, body);
			else if (editing) await templatesApi.update(editing.key, body);
			close(true);
			await Promise.all([refresh(), wellKnowns.refreshBoard()]);
		} catch (e) {
			error = explain(e);
		}
	}

	async function remove(template: PostitTemplate) {
		if (!confirm(m['postit_templates.confirm_delete']({ name: template.name }))) return;
		error = '';
		try {
			await templatesApi.remove(template.key);
			await Promise.all([refresh(), wellKnowns.refreshBoard()]);
		} catch (e) {
			error = explain(e);
		}
	}

	// A new one being written stays open; editing an existing one belongs to
	// the scope it was listed in.
	function show(next: Tab) {
		tab = next;
		if (editing !== 'new') editing = null;
		refresh();
	}

	function pickOrg(next: UUID) {
		org = next;
		if (editing !== 'new') editing = null;
		refresh();
	}

	// Where saving a new one puts it, in the tab's words.
	function where() {
		return tab === 'org' ? (orgs.byId(org)?.name ?? '') : m[`postit_templates.tab_${tab}`]();
	}

	refresh();
</script>

<Modal
	{onclose}
	withCloseButton={false}
	onaccept={onclose}
	acceptText={m['postit_templates.close']()}
	dialogClass={cn("max-w-2xl max-h-[90dvh] overflow-y-auto", scrollClass)}
>
	<h2 class="text-lg font-semibold">{m['postit_templates.title']()}</h2>

	<div class="flex gap-1 border-b border-main-border" role="tablist">
		{#each tabs as key (key)}
			<button
				type="button"
				role="tab"
				aria-selected={tab === key}
				class={cn(
					'-mb-px border-b-2 px-3 py-1 text-sm',
					tab === key ? 'border-primary font-semibold' : 'border-transparent opacity-70'
				)}
				onclick={() => show(key)}
			>
				{m[`postit_templates.tab_${key}`]()}
			</button>
		{/each}
	</div>
	{#if tab === 'org'}
		<label class="flex flex-col gap-1 text-sm">
			<span class="font-semibold">{m['postit_templates.org_pick']()}</span>
			<select
				class="rounded-md border border-main-border bg-background px-2 py-1"
				value={org}
				onchange={(e) => pickOrg(e.currentTarget.value)}
			>
				{#each orgs.list as o (o.id)}
					<option value={o.id}>{o.name}</option>
				{/each}
			</select>
		</label>
		{#if org !== orgId}
			<p class="text-xs opacity-70">{m['postit_templates.org_elsewhere']()}</p>
		{/if}
	{/if}
	<p class="text-xs opacity-70">{m[`postit_templates.hint_${tab}`]()}</p>

	{#if error}
		<p class="text-sm text-destructive">{error}</p>
	{/if}

	{#if loading}
		<p class="text-sm opacity-70">{m['postit_templates.loading']()}</p>
	{:else if list.length === 0}
		<p class="text-sm opacity-70">{m['postit_templates.empty']()}</p>
	{:else}
		<ul class="flex flex-col gap-2">
			{#each list as template (template.key)}
				<li
					class="flex items-center justify-between gap-2 rounded-md border border-main-border px-3 py-2"
				>
					<div class="flex min-w-0 flex-col">
						<span class="truncate text-sm font-semibold">{template.name}</span>
						<span class="truncate text-xs opacity-60">{template.resource}</span>
					</div>
					{#if canManage}
						<div class="flex shrink-0 gap-2">
							<Button
								variant="secondary"
								class="px-2 py-1 text-xs"
								onclick={() => startEdit(template)}
							>
								{m['postit_templates.edit']()}
							</Button>
							<Button
								variant="destructive"
								class="px-2 py-1 text-xs"
								onclick={() => remove(template)}
							>
								{m['postit_templates.delete']()}
							</Button>
						</div>
					{/if}
				</li>
			{/each}
		</ul>
	{/if}

	{#if !canManage}
		<p class="text-xs opacity-70">{m['postit_templates.read_only']()}</p>
		{#if editing === 'new'}
			<p class="text-xs opacity-70">{m['postit_templates.draft_kept']()}</p>
		{/if}
	{:else if editing === null}
		<Button variant="primary" class="self-start" onclick={startNew}>
			{m['postit_templates.new']()}
		</Button>
	{:else if editing === 'new'}
		<p class="text-sm font-semibold">{m['postit_templates.creating_in']({ where: where() })}</p>
		<PostitTemplateForm bind:draft={newDraft} onsave={save} oncancel={() => close(false)} />
	{:else}
		{#key editing}
			<PostitTemplateForm bind:draft={editDraft} onsave={save} oncancel={() => close(false)} />
		{/key}
	{/if}
</Modal>
