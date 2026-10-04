<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import logo from '$assets/logo.png';
	import Icon from '$components/Icon/Icon.svelte';
	import LocaleSwitcher from '$components/LocaleSwitcher/LocaleSwitcher.svelte';
	import UserMenu from '$components/UserMenu/UserMenu.svelte';

	import { SvelteFlowProvider } from '@xyflow/svelte';

	import { session } from '$modules/session.svelte';
	import { isSidebarOpen, toggleSidebar } from '$stores/sidebar';
	import { boardList, refreshBoards } from '$stores/boards';
	import { cn } from '$lib/utils';

	onMount(() => {
		refreshBoards().catch((err) => console.error('Failed to load boards', err));
	});

	// Signing out in another tab, or a session that could not be renewed.
	$effect(() => {
		if (session.status === 'anonymous') void goto(resolve('/login'));
	});

	let { children } = $props();

	const buttonClass = $derived(
		cn(
			'flex p-3 rounded-md text-white hover:bg-main-hover transition-colors h-fit',
			$isSidebarOpen ? 'bg-main-hover' : '',
			'cursor-pointer'
		)
	);
</script>

<SvelteFlowProvider>
	<div class="flex h-screen w-screen flex-col">
		<div class="z-100! flex flex-row justify-between border-b border-main-border bg-main px-2 py-1">
			<div class="flex flex-row items-center gap-4">
				<button type="button" onclick={() => toggleSidebar()} class={buttonClass}>
					<Icon name="menu" class="h-5 w-5 text-white" />
				</button>
				<a href={resolve('/')}><img src={logo} alt="logo" class="logo" /></a>
			</div>
			<div class="flex flex-row items-center gap-3">
				<LocaleSwitcher />
				<UserMenu />
			</div>
		</div>
		<div class="flex h-full w-screen flex-row">
			<div
				class="z-100! flex h-full w-14 max-w-14 flex-col gap-3 border-r border-sidebar-border bg-sidebar! p-2 transition-transform duration-300"
			>
				<div class="flex cursor-pointer flex-row items-center rounded-md bg-sidebar-hover p-2">
					<Icon name="graph" class="h-6 w-6 text-white!" />
				</div>
			</div>
			{#if $isSidebarOpen}
				<div class="z-100! flex w-50 flex-col gap-1 overflow-y-auto bg-sidebar-hover p-2">
					{#each $boardList as board (board.id)}
						<a
							href={resolve(`/board/${board.id}`)}
							class="truncate rounded-md border border-sidebar px-2 py-1.5 text-sm text-white transition-colors hover:bg-sidebar"
						>
							{board.name}
						</a>
					{/each}
				</div>
			{/if}
			<div class="flex flex-1 flex-col overflow-auto bg-background">
				<!-- Pages in (app) may assume a user; signing out unmounts them before it navigates. -->
				{#if session.user}
					{@render children()}
				{/if}
			</div>
		</div>
	</div>
</SvelteFlowProvider>

<style>
	.logo {
		width: 180px;
		height: auto;
	}
</style>
