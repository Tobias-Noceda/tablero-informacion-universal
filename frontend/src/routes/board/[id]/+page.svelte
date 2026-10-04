<script lang="ts">
	import './index.css';

	import type { Update } from '$modules/realtime.svelte.js';

	import Flow from './Flow.svelte';
	// import Realtime from './Realtime.svelte';

	import type { Node, Edge } from '@xyflow/svelte';
	// import type { PostIt, Strand } from '$types/api.js';

	let { data } = $props();
	const { boardId, nodes, edges, userId } = $derived(data) as {
		boardId: string
		boardName: string;
		nodes: Node[];
		edges: Edge[];
		userId: string;
	};

	function boardUpdate(update: Update) {
		data = {
			...data,
			boardName: update.board.name ?? data.boardName,
			nodes: update.board.postits ?? data.nodes,
			edges: update.board.strands ?? data.edges,
		};
	}
</script>

{#key boardId}
	<!-- <Realtime boardId={id} {boardUpdate}> -->
	<Flow {nodes} {edges} {boardId} {userId} {boardUpdate} />
	<!-- </Realtime> -->
{/key}
