<script lang="ts">
	import './index.css';

	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';

	import type { Node, Edge } from '@xyflow/svelte';
	import type { Update } from '$modules/realtime.svelte.js';
	import type { loadBoard } from '$lib/helpers/board-load';
	import { mapPostitWithTitle, type PostItWithTitle } from '$lib/helpers/post-it.js';
	import { landingPath } from '$lib/helpers/routes';

	import {
		isConfiguringBoard,
		isManagingCardTypes,
		isSharing,
		setConfiguringBoard,
		setManagingCardTypes,
		setSharing,
		setSidebar
	} from '$stores/sidebar';
	import { refreshBoards } from '$stores/boards';
	import { mobile } from '$stores/viewport.svelte';

	import ShareDialog from '$layouts/Share/ShareDialog.svelte';
	import PostitTemplatesPanel from '$layouts/PostitTemplates/PostitTemplatesPanel.svelte';
	import SideBar from '$layouts/Sidebar/SideBar.svelte';
	import Flow from './Flow.svelte';
	import { nodeType } from '$lib/well-knowns/one-off';
	import { strandToEdge } from '$lib/strands/style';
	import BoardConfiguration from '$layouts/BoardConfiguration/BoardConfiguration.svelte';
	import { deleteBoard, renameBoard } from '$services/board';

	let { data }: { data: Awaited<ReturnType<typeof loadBoard>> } = $props();

	const { boardId, orgId, role } = $derived(data);
	const nodes = $derived(
		data.nodes.map((postit) => mapPostitWithTitle(postit as PostItWithTitle)) as Node[]
	);
	const edges = $derived(data.edges as Edge[]);

	// Phones are read-only: the canvas and the sidebar act as for a viewer.
	// The share, configuration and card-type dialogs keep the real role.
	const viewRole = $derived(mobile.current ? ('viewer' as const) : role);

	// Viewers look: no cards to drop and no credentials (the backend refuses
	// all of it anyway).
	const canEdit = $derived(viewRole === 'owner' || viewRole === 'editor');

	$effect(() => {
		if (!canEdit) setSidebar(null);
	});

	function boardUpdate(update: Update) {
		data = {
			...data,
			boardName: update.board.name ?? data.boardName,
			nodes: (update.board.postits?.map((p) => ({ ...p, type: nodeType(p.type) })) ??
				data.nodes) as typeof data.nodes,
			edges: update.board.strands?.map(strandToEdge) ?? data.edges
		};
	}

	const onUpdateBoard = async (name: string) => {
		await renameBoard(data.boardId, name);
		refreshBoards();
		data = { ...data, boardName: name };
		setConfiguringBoard(false);
	}

	const onDeleteBoard = async () => {
		await deleteBoard(data.boardId);
		await refreshBoards();
		await goto(resolve(landingPath(orgId)));
		setConfiguringBoard(false);
	}

	async function left() {
		setSharing(false);
		await refreshBoards();
		await goto(resolve(landingPath(orgId)));
	}
</script>

<div class="flex flex-row flex-1 min-h-0">
	<!-- No board tools on a phone; Members is in the TopBar's drawer. -->
	<div class="hidden md:contents">
		<SideBar role={viewRole} {boardId} />
	</div>
	<div class="flex flex-col flex-1 min-h-0 overflow-auto bg-background">
		{#key boardId}
			<Flow {nodes} {edges} {role} readOnly={mobile.current} {boardId} {orgId} {boardUpdate} />
		{/key}
	</div>
</div>

{#if $isSharing}
	<ShareDialog board={boardId} {role} onclose={() => setSharing(false)} onleave={left} />
{/if}

{#if $isManagingCardTypes}
	<PostitTemplatesPanel {boardId} {orgId} {role} onclose={() => setManagingCardTypes(false)} />
{/if}

{#if $isConfiguringBoard}
	<BoardConfiguration
		boardName={data.boardName}
		role={role}
		onclose={() => setConfiguringBoard(false)}
		onedit={onUpdateBoard}
		ondelete={onDeleteBoard}
	/>
{/if}
