<script lang="ts">
	import { SvelteFlow, Controls, useSvelteFlow, type Node, type Edge } from '@xyflow/svelte';

	import { useDnD } from './DnDProvider.svelte';
	import Dock from './Dock.svelte';

	import * as postItsApi from '$services/post-it';
	import type { Board } from '$types/api';
	import Modal from '$components/Modal/Modal.svelte';
	import Input from '$components/Input/Input.svelte';
	import { nodesMap } from '$components/Nodes/node-map';
	import { mouses } from '$stores/mouses.svelte';

	let { nodes, edges, name, boardId }: {
		nodes: Node[],
		edges: Edge[],
		name: string,
		boardId: string,
	} = $props();

	$effect(() => console.log(nodes));

	let selectedNode: Node | null = $state(null);

	let creatingNode = $state<Board['postits'][number] | null>(null);
	let text = $state('');
	let keyword = $state('');
	// let latitude = $state('');
	// let longitude = $state('');
	// let start_date = $state('');
	// let end_date = $state('');

	const { screenToFlowPosition, flowToScreenPosition } = useSvelteFlow();

	mouses.updateMappers(screenToFlowPosition, flowToScreenPosition);

	const type = useDnD();

	const onDragOver = (event: DragEvent) => {
		event.preventDefault();

		if (event.dataTransfer) {
			event.dataTransfer.dropEffect = 'move';
		}
	};

	const onDrop = async (event: DragEvent) => {
		event.preventDefault();

		if (!type.current) {
			return;
		}

		const position = screenToFlowPosition({
			x: event.clientX,
			y: event.clientY
		});

		if (type.current === 'dog_facts' || type.current === 'dolar_oficial') {
			const newPostIt = await postItsApi.create_well_known(boardId, type.current, {});
			await postItsApi.move(newPostIt.id, position.x, position.y);
			nodes = [...nodes, { id: newPostIt.id, position, type: type.current } as Node];
		}
		creatingNode = {
			id: crypto.randomUUID(),
			type: type.current,
			position
		};
	};

	const createNode = async () => {
		if (!creatingNode) return;

		if (creatingNode.type === 'static_card' && !text.trim()) return;
		if (creatingNode.type === 'events_search' && !keyword.trim()) return;

		const params: Record<string, string> = creatingNode.type === 'static_card' ? { text: text.trim() } : { "$keyword": keyword.trim() };

		const newPostIt = await postItsApi.create_well_known(boardId, creatingNode.type!, params);
		await postItsApi.move(newPostIt.id, creatingNode.position.x, creatingNode.position.y);

		const newNode = {
			...creatingNode,
			data: { text, keyword }
		};

		nodes = [...nodes, newNode];
		creatingNode = null;
		text = '';
		keyword = '';
	};
</script>

<div class="flex flex-row h-full w-full">
	<main class="dndflow">
		<h1 class="text-2xl font-bold mb-4 ml-3">{name}</h1>
		<div class="reactflow-wrapper">
			<SvelteFlow
				bind:nodes
				bind:edges
				nodeTypes={nodesMap}
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
		<div
			class="flex flex-col gap-2 p-4 bg-tertiary border-l border-tertiary-border rounded-l-2xl w-70 text-tertiary-text"
		>
			<h2>Selected Node</h2>
			<p>ID: {selectedNode.id}</p>
			<p>Type: {selectedNode.type}</p>
			<p>Position: ({selectedNode.position.x}, {selectedNode.position.y})</p>
		</div>
	{/if}

	{#if creatingNode}
		<Modal
			onclose={() => (creatingNode = null)}
			onaccept={() => {
				if (creatingNode) {
					createNode();
				}
			}}
			acceptText="Create"
			acceptDisabled={(creatingNode.type === 'default' && text.trim() === '') ||
				(creatingNode.type === 'events_search' && keyword.trim() === '')}
		>
			<h2 class="text-lg font-semibold">Create Node</h2>
			{#if creatingNode.type === 'static_card'}
				<Input label="Text" placeholder="Enter text" bind:value={text} />
			{/if}
			{#if creatingNode.type === 'events_search'}
				<Input label="Keyword" placeholder="Enter keyword" bind:value={keyword} />
			{/if}
		</Modal>
	{/if}
</div>

<style>
	main.dndflow {
		display: flex;
		flex-direction: column;
		padding: 10px;
	}
</style>
