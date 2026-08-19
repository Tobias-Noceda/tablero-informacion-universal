<script lang="ts">
	import { SvelteFlow, Controls, useSvelteFlow, type Node, type Edge, ConnectionMode, type Connection } from '@xyflow/svelte';

	import { useDnD } from './DnDProvider.svelte';
	import Dock from './Dock.svelte';

	import * as postItsApi from '$services/post-it';
	import * as edgesApi from '$services/edge';
	import type { Board } from '$types/api';
	import Modal from '$components/Modal/Modal.svelte';
	import Input from '$components/Input/Input.svelte';
	import { nodesMap, parameters } from '$components/Nodes/node-map';
	import { edgesMap } from '$components/Edges/edge-map';
	import Button from '$components/Button/Button.svelte';

	let { nodes, edges, name, boardId }: {
		nodes: Node[],
		edges: Edge[],
		name: string,
		boardId: string,
	} = $props();

	let selectedNode: Node | null = $state(null);
	let selectedEdge: Edge | null = $state(null);

	let creatingNode = $state<Board['postits'][number] | null>(null);
	let paramValues = $state<Record<string, string>>({});

	const creatingParams = $derived(creatingNode ? (parameters[creatingNode.type!] ?? []) : []);
	const missingRequired = $derived(
		creatingParams.some((p) => p.default === undefined && !(paramValues[p.key] ?? '').trim())
	);

	const { screenToFlowPosition } = useSvelteFlow();

	const type = useDnD();

	// Board handlers
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

	const onBoardClick = () => {
		selectedNode = null;
		selectedEdge = null;
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

	// Node handlers
	const onNodeClick = async (event: { event: MouseEvent | TouchEvent, node: Node }) => {
		event.event.preventDefault();
		event.event.stopPropagation();
		if (selectedNode?.id === event.node.id) {
			selectedNode = null;
			nodes = nodes.map((n) => { return { ...n, data: { ...n.data, isSelected: false } } });
		} else {
			selectedNode = event.node;
			nodes = nodes.map((n) => { return { ...n, data: { ...n.data, isSelected: n.id === event.node.id } } });
		}
	};

	const onNodeDragStop = async (event: { targetNode: Node | null, nodes: Node[], event: MouseEvent | TouchEvent }) => {
		const node = event.targetNode;
		if (!node) return;
		await postItsApi.move(node.id, node.position.x, node.position.y);
	};

	// Edge handlers
	const onEdgeClick = async (event: {event: MouseEvent, edge: Edge}) => {
		event.event.preventDefault();
		event.event.stopPropagation();
		if (selectedEdge?.target === event.edge.target && selectedEdge?.source === event.edge.source) {
			selectedEdge = null;
			edges = edges.map((e) => { return { ...e, data: { ...e.data, isSelected: false } } });
		} else {
			selectedEdge = event.edge;
			edges = edges.map((e) => { return { ...e, data: { ...e.data, isSelected: e.id === event.edge.id } } });
		}
	};

	const onConnect = async (connection: Connection) => {
		// console.log('onConnect', connection);
		const newEdge = await edgesApi.connect(boardId, connection.source, connection.target)
			.then(() => ({ id: crypto.randomUUID(), source: connection.source, target: connection.target })); 
		edges = [...edges, { id: newEdge.id, source: connection.source, target: connection.target }];
	};

	// Keyboard shortcuts
	const onKeyDown = (event: KeyboardEvent) => {
		if (event.key === 'Delete' || event.key === 'Backspace') {
			console.log('Deleting edge', selectedEdge);
			if (selectedEdge) {
				edgesApi.disconnect(boardId, selectedEdge.source, selectedEdge.target);
				edges = edges.filter((e) => e.id !== selectedEdge?.id);
				selectedEdge = null;
			} else if (selectedNode) {
				postItsApi.del(selectedNode.id);
				nodes = nodes.filter((n) => n.id !== selectedNode?.id);
				selectedNode = null;
			}
		}
	};

	// Effects
	$effect(() => {
		window.addEventListener('keydown', onKeyDown);
		return () => window.removeEventListener('keydown', onKeyDown);
	});

	$effect(() => {
		if (selectedNode) {
			selectedEdge = null;
		}
	});

	$effect(() => {
		if (selectedEdge) {
			selectedNode = null;
		}
	});

	$effect(() => {
		if (creatingNode) {
			selectedNode = null;
			selectedEdge = null;
		}
	});
</script>

<div class="flex flex-row h-full w-full">
	<main class="dndflow">
		<h1 class="text-2xl font-bold mb-4 ml-3">{name}</h1>
		<div class="reactflow-wrapper">
			<SvelteFlow
				bind:nodes
				bind:edges
				nodeTypes={nodesMap}
				edgeTypes={edgesMap}
				defaultEdgeOptions={{ type: 'floating' }}
				fitView
				connectionMode={ConnectionMode.Loose}
				ondragover={onDragOver}
				ondrop={onDrop}
				onnodeclick={onNodeClick}
				onnodedragstop={onNodeDragStop}
				onedgeclick={onEdgeClick}
				onconnect={onConnect}
				onpaneclick={onBoardClick}
				colorMode="system"
				class="bg-transparent!"
				title="Board Flow"
				attributionPosition={undefined}
			>
				<Controls />
			</SvelteFlow>
		</div>
		<Dock />
	</main>
	{#if selectedNode}
		<div
			class="flex flex-col p-4 bg-tertiary border-l border-tertiary-border rounded-l-2xl w-70 h-full justify-between text-tertiary-text"
		>
			<div class="flex flex-col gap-2">
				<h2 class="text-lg font-semibold">Selected Node</h2>
				<p>ID: {selectedNode.id}</p>
				<p>Type: {selectedNode.type}</p>
				<p>Position: ({selectedNode.position.x}, {selectedNode.position.y})</p>
			</div>
			<Button
				variant="destructive"
				onclick={() => {
					postItsApi.del(selectedNode!.id);
					nodes = nodes.filter((n) => n.id !== selectedNode?.id);
					selectedNode = null;
				}}
			>
				Delete Node
			</Button>
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

	:global(.svelte-flow__attribution) {
		display: none;
	}
</style>
