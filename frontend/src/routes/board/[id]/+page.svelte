<script lang="ts">
	import './index.css';

	import type { Update } from '$modules/realtime.svelte.js';

	import Flow from './Flow.svelte';
	import DnDProvider from './DnDProvider.svelte';
	import MouseTracker from './MouseTracker.svelte';

	import { page } from '$app/state';
	import type { Node, Edge } from '@xyflow/svelte';

	const id = page.params.id!;

	const { data } = $props();
	const { nodes, edges, name } = $derived(data) as {
		nodes: Node[];
		edges: Edge[];
		name: string;
	};

	function boardUpdate(update: Update) {
		data.nodes = update.board.postits;
		data.edges = update.board.strands;
		data.name = update.board.name;
	}
</script>

<DnDProvider>
	<MouseTracker boardId={id} {boardUpdate}>
		<Flow {name} {nodes} {edges} boardId={id} />
	</MouseTracker>
</DnDProvider>
