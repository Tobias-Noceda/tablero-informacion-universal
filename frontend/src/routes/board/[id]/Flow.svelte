<script lang="ts">
	import { SvelteFlow, Controls, useSvelteFlow, type Node } from '@xyflow/svelte';

	import { useDnD } from './DnDProvider.svelte';
	import Dock from './Dock.svelte';

	import * as boardApi from '$services/board';
	import * as postItsApi from '$services/post-it';

	const { boardId } = $props();

	// const nodes = await

	let selectedNode: Node | null = $state(null);

	let nodes = $state.raw([
		{
			id: '1',
			type: 'input',
			data: { label: 'Input Node' },
			position: { x: 150, y: 5 }
		},
		{
			id: '2',
			type: 'default',
			data: { label: 'Default Node' },
			position: { x: 0, y: 150 }
		},
		{
			id: '3',
			type: 'output',
			data: { label: 'Output Node' },
			position: { x: 300, y: 150 }
		}
	]);

	let edges = $state.raw([
		{
			id: '1-2',
			type: 'default',
			source: '1',
			target: '2'
		},
		{
			id: '1-3',
			type: 'smoothstep',
			source: '1',
			target: '3'
		}
	]);

	const { screenToFlowPosition } = useSvelteFlow();

	const type = useDnD();

	const onDragOver = (event: DragEvent) => {
		event.preventDefault();

		if (event.dataTransfer) {
			event.dataTransfer.dropEffect = 'move';
		}
	};

	const onDrop = (event: DragEvent) => {
		event.preventDefault();

		if (!type.current) {
			return;
		}

		const position = screenToFlowPosition({
			x: event.clientX,
			y: event.clientY
		});

		// API interaction
		const newNode = {
			id: `${Math.random()}`,
			type: type.current,
			position,
			data: { label: `${type.current} node` },
			origin: [0.5, 0.0]
		} satisfies Node;

		nodes = [...nodes, newNode];
	};
</script>

<div class="flex flex-row h-full w-full">
	<main class="dndflow">
		<div class="reactflow-wrapper">
			<SvelteFlow
				bind:nodes
				bind:edges
				fitView
				ondragover={onDragOver}
				ondrop={onDrop}
				onnodeclick={(event) => {
					if (event.node.id === selectedNode?.id) {
						selectedNode = null;
					} else {
						selectedNode = event.node;
					}
				}}
				colorMode="system"
				class="bg-transparent!"
			>
				<Controls />
			</SvelteFlow>
		</div>
		<Dock />
	</main>
	{#if selectedNode}
		<div class="flex flex-col gap-2 p-4 bg-tertiary border-l border-tertiary-border rounded-l-2xl w-70">
			<h2>Selected Node</h2>
			<p>ID: {selectedNode.id}</p>
			<p>Type: {selectedNode.type}</p>
			<p>Position: ({selectedNode.position.x}, {selectedNode.position.y})</p>
		</div>
	{/if}
</div>

<style>
	main.dndflow {
		display: flex;
		flex-direction: column;
		padding: 10px;
	}
</style>
