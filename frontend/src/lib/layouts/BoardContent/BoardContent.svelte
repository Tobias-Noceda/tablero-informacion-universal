<script lang="ts">
	import './index.css';

	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import type { Node, Edge } from '@xyflow/svelte';
	import type { Update } from '$modules/realtime.svelte.js';
	import type { loadBoard } from '$lib/helpers/board-load';
	import { mapPostitWithTitle, type PostItWithTitle } from '$lib/helpers/post-it.js';
	import { landingPath } from '$lib/helpers/routes';

	import { isSharing, setSharing, setSidebar } from '$stores/sidebar';
	import { refreshBoards } from '$stores/boards';

	import ShareDialog from '$components/Share/ShareDialog.svelte';
	import SideBar from '$layouts/Sidebar/SideBar.svelte';
	import Flow from './Flow.svelte';

	let { data }: { data: Awaited<ReturnType<typeof loadBoard>> } = $props();

	const { boardId, orgId, role } = $derived(data);
	const nodes = $derived(
		data.nodes.map((postit) => mapPostitWithTitle(postit as PostItWithTitle)) as Node[]
	);
	const edges = $derived(data.edges as Edge[]);

	// Viewers look: no cards to drop and no credentials (the backend refuses
	// all of it anyway).
	const canEdit = $derived(role === 'owner' || role === 'editor');

	$effect(() => {
		if (!canEdit) setSidebar(null);
	});

	// On a phone an open card takes the whole board, its sidebar included.
	const cardOpen = $derived(!!page.params.nodeId);

	function boardUpdate(update: Update) {
		data = {
			...data,
			boardName: update.board.name ?? data.boardName,
			nodes: (update.board.postits ?? data.nodes) as typeof data.nodes,
			edges: update.board.strands ?? data.edges
		};
	}

	async function left() {
		setSharing(false);
		await refreshBoards();
		await goto(resolve(landingPath(orgId)));
	}
</script>

<div class="flex flex-row flex-1 min-h-0">
	<div class={cardOpen ? 'hidden md:contents' : 'contents'}>
		<SideBar {canEdit} {boardId} />
	</div>
	<div class="flex flex-col flex-1 min-h-0 overflow-auto bg-background">
		{#key boardId}
			<Flow {nodes} {edges} {role} {boardId} {orgId} {boardUpdate} />
		{/key}
	</div>
</div>

{#if $isSharing}
	<ShareDialog board={boardId} {role} onclose={() => setSharing(false)} onleave={left} />
{/if}
