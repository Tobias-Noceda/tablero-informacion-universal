<script lang="ts">
	import { onMount } from 'svelte';

	import { SvelteFlowProvider } from '@xyflow/svelte';

	import { session } from '$modules/session.svelte';
	import { refreshBoards } from '$stores/boards';
	import { orgs } from '$stores/org.svelte';
	import { resolve } from '$app/paths';
	import { goto } from '$app/navigation';
	import DnDProvider from '$providers/DnDProvider.svelte';
	import TopBar from '$layouts/TopBar/TopBar.svelte';

	onMount(() => {
		refreshBoards().catch((err) => console.error('Failed to load boards', err));
		orgs.ensure().catch((err) => console.error('Failed to load organizations', err));
	});

	// Signing out in another tab, or a session that could not be renewed.
	$effect(() => {
		if (session.status === 'anonymous') void goto(resolve('/login'));
	});

	let { children } = $props();
</script>

<SvelteFlowProvider>
	<div class="flex flex-col h-screen w-screen">
		<TopBar />
		<div class="flex flex-col flex-1 min-h-0 overflow-auto bg-background">
			<!-- Pages in (app) may assume a user; signing out unmounts them before it navigates. -->
			{#if session.user}
				<DnDProvider>
					{@render children()}
				</DnDProvider>
			{/if}
		</div>
	</div>
</SvelteFlowProvider>
