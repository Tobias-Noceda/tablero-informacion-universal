<script lang="ts">
	import { onMount } from 'svelte';

	import { getSidebar, setSidebar } from '$stores/sidebar';
	import { refreshBoards } from '$stores/boards';

	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import CardsSidebar from '$layouts/Sidebar/CardsSidebar.svelte';
	import { cn } from '$lib/utils';
	import Icon from '$components/Icon/Icon.svelte';
	import CredentialsSidebar from '$layouts/Sidebar/CredentialsSidebar.svelte';
	import ShareDialog from '$components/Share/ShareDialog.svelte';
	import { page } from '$app/state';
	import type { BoardRole } from '$types/api';

	const { children } = $props();

	const boardId = $derived(page.params.id!);
	const role = $derived((page.data.role ?? 'viewer') as BoardRole);

	// Viewers look: no cards to drop and no credentials (the backend refuses
	// all of it anyway).
	const canEdit = $derived(role === 'owner' || role === 'editor');
	let sharing = $state(false);

	$effect(() => {
		if (!canEdit) setSidebar(null);
	});

	async function left() {
		sharing = false;
		await refreshBoards();
		await goto(resolve('/'));
	}

	onMount(() => {
		refreshBoards().catch((err) => console.error('Failed to load boards', err));
	});
</script>

<div class="flex flex-row flex-1 min-h-0">
	<div
		class="flex flex-col justify-between bg-sidebar! border-r border-sidebar-border transition-transform duration-300 h-full p-2 w-14 max-w-14 z-100!"
	>
		{#snippet SidebarButton(iconName: string, isActive: boolean, onClick: () => void)}
			<button
				class={cn(
					'flex flex-row items-center rounded-md bg-sidebar hover:bg-sidebar-hover cursor-pointer p-2',
					isActive ? 'bg-sidebar-hover' : ''
				)}
				onclick={onClick}
			>
				<Icon name={iconName} class="w-6 h-6 text-main-text!" />
			</button>
		{/snippet}
		<div class="flex flex-col gap-3">
			{#if canEdit}
				{@render SidebarButton('card-stack', $getSidebar === 'cards', () =>
					setSidebar($getSidebar === 'cards' ? null : 'cards')
				)}
				{@render SidebarButton('key', $getSidebar === 'credentials', () =>
					setSidebar($getSidebar === 'credentials' ? null : 'credentials')
				)}
			{/if}
		</div>
		<div class="flex flex-col gap-3">
			{@render SidebarButton('users', sharing, () => (sharing = true))}
			{@render SidebarButton('configuration', false, () => console.log('Manage board settings'))}
		</div>
	</div>
	{#if $getSidebar !== null}
		<aside>
			{#if $getSidebar === 'cards'}
				<CardsSidebar />
			{:else if $getSidebar === 'credentials'}
				{#key boardId}
					<CredentialsSidebar board={boardId} />
				{/key}
			{/if}
		</aside>
	{/if}
	<div class="flex flex-col flex-1 overflow-auto bg-background">
		{@render children()}
	</div>
</div>

{#if sharing}
	<ShareDialog board={boardId} {role} onclose={() => (sharing = false)} onleave={left} />
{/if}

<style>
	aside {
		display: flex;
		flex-direction: column;
		background: var(--color-sidebar-hover);
		padding: 8px;
		width: 200px;
		min-height: 0;
		height: 100%;
		overflow-y: auto;
		z-index: 100 !important;
	}
</style>
