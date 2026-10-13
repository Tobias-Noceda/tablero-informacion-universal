<script lang="ts">
	import './index.css';

	import type { Update } from '$modules/realtime.svelte.js';

	import Flow from './Flow.svelte';
	// import Realtime from './Realtime.svelte';

	import type { Node, Edge } from '@xyflow/svelte';
	import { mapPostitWithTitle, type PostItWithTitle } from '$lib/helpers/post-it.js';
	// import type { PostIt, Strand } from '$types/api.js';

	// type PageProps = { data: Board };
	let { data } = $props();
	const { boardId, nodes, edges, userId } = $derived({
		...data,
		nodes: data.nodes.map(mapPostitWithTitle)
	}) as {
		boardId: string
		boardName: string;
		nodes: Node[];
		edges: Edge[];
		userId: string;
	};

	function boardUpdate(update: Update) {
		console.log('boardUpdate: ', update);
		data = {
			...data,
			boardName: update.board.name ?? data.boardName,
			nodes: update.board.postits?.map((postit) => mapPostitWithTitle(postit as PostItWithTitle)) ?? data.nodes,
			edges: update.board.strands ?? data.edges,
		};
	}
</script>

{#key boardId}
	<!-- <Realtime boardId={id} {boardUpdate}> -->
	<Flow {nodes} {edges} {boardId} {userId} {boardUpdate} />
	<!-- </Realtime> -->
{/key}
