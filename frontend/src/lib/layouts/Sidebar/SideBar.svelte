<script lang="ts">
    import { getSidebar, isSharing, setSharing, setSidebar } from '$stores/sidebar';
    import { cn } from '$lib/utils';
    import Icon from '$components/Icon/Icon.svelte';
    import CardsSidebar from './content/CardsSidebar.svelte';
    import CredentialsSidebar from './content/CredentialsSidebar.svelte';

    const { canEdit, boardId } = $props();
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
        {@render SidebarButton('users', $isSharing, () => setSharing(true))}
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
