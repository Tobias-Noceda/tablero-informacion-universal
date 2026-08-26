<script lang="ts">
	import { SvelteFlow, Controls, useSvelteFlow, type Node, type Edge } from '@xyflow/svelte';

	import { useDnD } from './DnDProvider.svelte';
	import Dock from './Dock.svelte';

	import * as postItsApi from '$services/post-it';
	import type { Board } from '$types/api';
	import Modal from '$components/Modal/Modal.svelte';
	import Input from '$components/Input/Input.svelte';
	import { nodesMap, parameters } from '$components/Nodes/node-map';

	let { nodes, edges, name, boardId }: {
		nodes: Node[],
		edges: Edge[],
		name: string,
		boardId: string,
	} = $props();

	$effect(() => console.log(nodes));

	let selectedNode: Node | null = $state(null);

	let creatingNode = $state<Board['postits'][number] | null>(null);
	let paramValues = $state<Record<string, string>>({});

	const creatingParams = $derived(creatingNode ? (parameters[creatingNode.type!] ?? []) : []);
	const missingRequired = $derived(
		creatingParams.some((p) => p.default === undefined && !(paramValues[p.key] ?? '').trim())
	);

	const { screenToFlowPosition } = useSvelteFlow();

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

		if ((parameters[type.current] ?? []).length === 0) {
			const newPostIt = await postItsApi.create_well_known(boardId, type.current, {});
			await postItsApi.move(newPostIt.id, position.x, position.y);
			nodes = [...nodes, { id: newPostIt.id, position, type: type.current } as Node];
			return;
		}

		paramValues = Object.fromEntries(
			(parameters[type.current] ?? []).map((p) => [p.key, p.default ?? ''])
		);
		creatingNode = {
			id: crypto.randomUUID(),
			type: type.current,
			position
		};
	};

	const createNode = async () => {
		if (!creatingNode || missingRequired) return;

		const params = Object.fromEntries(
			creatingParams.map((p) => [p.key, (paramValues[p.key] ?? '').trim()])
		);

		const newPostIt = await postItsApi.create_well_known(boardId, creatingNode.type!, params);
		await postItsApi.move(newPostIt.id, creatingNode.position.x, creatingNode.position.y);

		nodes = [...nodes, { ...creatingNode, id: newPostIt.id, data: params }];
		creatingNode = null;
		paramValues = {};
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
			acceptDisabled={missingRequired}
		>
			<h2 class="text-lg font-semibold">Create Node</h2>
			{#each creatingParams as param (param.key)}
				<Input
					label={param.label}
					placeholder={param.placeholder}
					type={param.type === 'number' ? 'number' : 'text'}
					required={param.default === undefined}
					bind:value={paramValues[param.key]}
				/>
			{/each}
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
