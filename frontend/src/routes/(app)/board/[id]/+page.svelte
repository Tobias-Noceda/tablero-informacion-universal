<script lang="ts">
	import './index.css';

	import type { Update } from '$modules/realtime.svelte.js';

	import Flow from './Flow.svelte';
	import DnDProvider from './DnDProvider.svelte';

	import { page } from '$app/state';
	import type { Node, Edge } from '@xyflow/svelte';
	import type { BoardRole } from '$types/api';

	const id = $derived(page.params.id!);

	let { data } = $props();
	const { nodes, edges, name, role } = $derived(data) as {
		nodes: Node[];
		edges: Edge[];
		name: string;
		role: BoardRole;
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
		<Flow {name} {nodes} {edges} {role} boardId={id} {boardUpdate} />
	</DnDProvider>
{/key}
