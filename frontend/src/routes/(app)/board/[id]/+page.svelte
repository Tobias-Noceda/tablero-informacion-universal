<script lang="ts">
	import './index.css';

	import type { Update } from '$modules/realtime.svelte.js';

	import Flow from './Flow.svelte';

	import type { Node, Edge } from '@xyflow/svelte';
	import type { BoardRole } from '$types/api';
	import { mapPostitWithTitle, type PostItWithTitle } from '$lib/helpers/post-it.js';

	let { data } = $props();
	const { boardId, nodes, edges, role } = $derived({
		...data,
		nodes: data.nodes.map((postit) => mapPostitWithTitle(postit as PostItWithTitle))
	}) as {
		boardId: string;
		boardName: string;
		nodes: Node[];
		edges: Edge[];
		role: BoardRole;
	};

	function boardUpdate(update: Update) {
		data = {
			...data,
			boardName: update.board.name ?? data.boardName,
			nodes:
				update.board.postits?.map((postit) => mapPostitWithTitle(postit as PostItWithTitle)) ??
				data.nodes,
			edges: update.board.strands ?? data.edges
		};
	}
</script>

{#key boardId}
	<Flow {nodes} {edges} {role} {boardId} {boardUpdate} />
{/key}
