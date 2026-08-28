<script lang="ts">
	import './index.css';

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
</script>

<DnDProvider>
	<MouseTracker boardId={id}>
		<Flow {name} {nodes} {edges} boardId={id} />
	</MouseTracker>
</DnDProvider>
