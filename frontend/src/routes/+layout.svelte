<script lang="ts">
	import { onMount } from 'svelte';
	import { locales, type Locale } from '$lib/paraglide/runtime';
	import './layout.css';
	import favicon from '$assets/favicon.svg';
	import logo from '$assets/logo.png';
	import Icon from '$components/Icon/Icon.svelte';

 	import { SvelteFlowProvider } from '@xyflow/svelte';

	import { setLocale, getLocale } from '$lib/paraglide/runtime';
	import { isSidebarOpen, toggleSidebar } from '$stores/sidebar';
	import { boardList, refreshBoards } from '$stores/boards';
	import { cn } from '$lib/utils';

	onMount(() => {
		refreshBoards().catch((err) => console.error('Failed to load boards', err));
	});

	let { children } = $props();

	const buttonClass = $derived(cn(
		"flex p-3 rounded-md text-white hover:bg-main-hover transition-colors h-fit",
		$isSidebarOpen ? "bg-main-hover" : "",
		"cursor-pointer"
	));

	const languageClass = (locale: Locale) => cn(
		"px-4 py-2 rounded-md text-white hover:bg-main-hover transition-colors text-main-text text-sm font-medium",
		getLocale() === locale ? "bg-main-hover" : "",
		"cursor-pointer"
	);
</script>

<svelte:head><link rel="icon" href={favicon} /></svelte:head>

<SvelteFlowProvider>
	<div class="flex flex-col h-screen w-screen">
		<div class="flex flex-row bg-main justify-between border-b border-main-border px-2 py-1">
			<div class="flex flex-row items-center gap-4">
				<button onclick={() => toggleSidebar()} class={buttonClass}>
					<Icon name="menu" class="w-5 h-5 text-white" />
				</button>
				<img src={logo} alt="logo" class="logo" />
			</div>
			<div class="flex flex-row items-center gap-3">
				{#each locales as locale (locale)}
					<button class={languageClass(locale)} onclick={() => setLocale(locale)}>{locale.toUpperCase()}</button>
				{/each}
			</div>
		</div>
		<div class="flex flex-row h-full w-screen">
			<div class="flex flex-col bg-sidebar! border-r border-sidebar-border transition-transform duration-300 h-full p-2 gap-3 w-14 max-w-14">
				<div class="flex flex-row items-center rounded-md bg-sidebar-hover cursor-pointer p-2">
					<Icon name="graph" class="w-6 h-6 text-white!" />
					<!-- <span class="text-white text-sm font-medium">Home</span> -->
				</div>
			</div>
			{#if $isSidebarOpen}
				<div class="flex flex-col bg-sidebar-hover p-2 w-50 gap-1 overflow-y-auto">
					{#each $boardList as board (board.id)}
						<a
							href="/board/{board.id}"
							class="px-2 py-1.5 rounded-md text-white text-sm hover:bg-main-hover transition-colors truncate"
						>
							{board.name}
						</a>
					{/each}
				</div>
			{/if}
			<div class="flex flex-col flex-1 overflow-auto bg-background">
				{@render children()}
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
