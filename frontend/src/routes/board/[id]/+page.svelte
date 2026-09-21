<script lang="ts">
	import './index.css';

	import type { Update } from '$modules/realtime.svelte.js';

	import Flow from './Flow.svelte';
	import DnDProvider from './DnDProvider.svelte';
	// import Realtime from './Realtime.svelte';

	import { page } from '$app/state';
	import type { Node, Edge } from '@xyflow/svelte';

	const id = $derived(page.params.id!);

	let { data } = $props();
	const { nodes, edges, name, userId } = $derived(data) as {
		nodes: Node[];
		edges: Edge[];
		name: string;
		userId: string;
	};

	function boardUpdate(update: Update) {
		data = {
			...data,
			nodes: update.board.postits ?? data.nodes,
			edges: update.board.strands ?? data.edges,
			name: update.board.name ?? data.name
		};
	}
</script>

{#key id}
	<DnDProvider>
		<!-- <Realtime boardId={id} {boardUpdate}> -->
		<Flow {name} {nodes} {edges} boardId={id} {userId} {boardUpdate} />
		<!-- </Realtime> -->
	</DnDProvider>
{/key}
