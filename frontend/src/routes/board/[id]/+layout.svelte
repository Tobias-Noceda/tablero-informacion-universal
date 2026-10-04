<script lang="ts">
	import { onMount } from 'svelte';

	import { getSidebar, setSidebar } from '$stores/sidebar';
	import { /* boardList, */ refreshBoards } from '$stores/boards';

	// import { resolve } from '$app/paths';

	import CardsSidebar from '$layouts/sidebar/CardsSidebar.svelte';
	import { cn } from '$lib/utils';
	import Icon from '$components/Icon/Icon.svelte';
	import CredentialsSidebar from '$layouts/sidebar/CredentialsSidebar.svelte';
	import { page } from '$app/state';

	const { children } = $props();

	const boardId = page.params.id;

	onMount(() => {
		refreshBoards().catch((err) => console.error('Failed to load boards', err));
	});
</script>

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
		{@render SidebarButton('card-stack', $getSidebar === 'cards', () => setSidebar($getSidebar === 'cards' ? null : 'cards'))}
		{@render SidebarButton('key', $getSidebar === 'credentials', () => setSidebar($getSidebar === 'credentials' ? null : 'credentials'))}
	</div>
	<div class="flex flex-col gap-3">
		{@render SidebarButton('users', false, () => console.log('Manage collaborators'))}
		{@render SidebarButton('configuration', false, () => console.log('Manage board settings'))}
	</div>
</div>
{#if $getSidebar !== null}
	<aside>
		{#if $getSidebar === 'cards'}
			<CardsSidebar />
		{:else if $getSidebar === 'credentials'}
			<!-- <div class="flex flex-col bg-sidebar-hover p-2 w-50 gap-1 overflow-y-auto z-100!">
            {#each $boardList as board (board.id)}
                <a
                    href={resolve(`/board/${board.id}`)}
                    class="px-2 py-1.5 rounded-md text-main-text text-sm hover:bg-sidebar border border-sidebar transition-colors truncate"
                >
                    {board.name}
                </a>
            {/each}
        </div> -->
			<CredentialsSidebar board={boardId!} />
		{/if}
	</aside>
{/if}
<div class="flex flex-col flex-1 overflow-auto bg-background">
	{@render children()}
</div>

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
