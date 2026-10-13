<script lang="ts">
	import { SvelteFlow, Controls, useSvelteFlow, type Node, type Edge, ConnectionMode, type Connection } from '@xyflow/svelte';

	import { useDnD } from '../../../lib/providers/DnDProvider.svelte';

	import * as postItsApi from '$services/post-it';
	import * as edgesApi from '$services/edge';
	import type { Board } from '$types/api';
	import Button from '$components/Button/Button.svelte';
	import SecretsPanel from '$layouts/Secrets/SecretsPanel.svelte';
	import * as secretsApi from '$services/secrets';
	import type { SecretMeta } from '$types/api';
	import { uuid } from '$lib/utils';
	import { nodesMap, parameters } from '$components/Nodes/node-map';
	import { edgesMap } from '$components/Edges/edge-map';
	import { mouses } from '$stores/mouses.svelte';
	import Realtime from './Realtime.svelte';
	import type { Update } from '$modules/sockets.svelte';
	import { isManagingSecrets, setManagingSecrets } from '$stores/sidebar';

	import { getUser } from '$stores/user';
	import NodeCreationPanel from '$layouts/NodeCreation/NodeCreationPanel.svelte';

	let { nodes, edges, boardId, userId, boardUpdate }: {
		nodes: Node[],
		edges: Edge[],
		boardId: string,
		userId: string,
		boardUpdate: (update: Update) => void
	} = $props();

	let selectedNode: Node | null = $state(null);
	let selectedEdge: Edge | null = $state(null);

	// Everything this user may bind on this board, wherever it lives.
	let usableSecrets = $state<SecretMeta[]>([]);

	// Refreshed whenever the panel closes, so a credential added there is
	// immediately pickable when creating a node.
	$effect(() => {
		if ($isManagingSecrets) return;
		secretsApi.usable(boardId, $getUser?.id ?? '').then((s) => (usableSecrets = s)).catch(() => (usableSecrets = []));
	});
	let creatingNode = $state<Board['postits'][number] | null>(null);
	let paramValues = $state<Record<string, string>>({});

	const { screenToFlowPosition, flowToScreenPosition } = useSvelteFlow();

	mouses.updateMappers(screenToFlowPosition, flowToScreenPosition);

	const type = useDnD();

	const uuidRegex = /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

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

		// if ((parameters[type.current] ?? []).length === 0) {
		// 	const newPostIt = await postItsApi.create_well_known(boardId, type.current, {}, $getUser?.id ?? '');
		// 	await postItsApi.move(newPostIt.id, position.x, position.y);
		// 	nodes = [...nodes, { id: newPostIt.id, position, type: type.current } as Node];
		// 	return;
		// }

		paramValues = Object.fromEntries(
			(parameters[type.current] ?? []).map((p) => [p.key, p.default ?? ''])
		);
		creatingNode = {
			id: uuid(),
			type: type.current,
			position,
		};
	};

	const onBoardClick = () => {
		selectedNode = null;
		selectedEdge = null;

		nodes = nodes.map((n) => {
			return { ...n, data: { ...n.data, isSelected: false } };
		});
		edges = edges.map((e) => {
			return { ...e, data: { ...e.data, isSelected: false } };
		});
	};

	const createNode = (newNode: Node) => {
		nodes = [...nodes, { ...newNode }];
		creatingNode = null;
		paramValues = {};
	};

	const deleteNode = async (node: Node) => {
		await postItsApi.del(node.id)
			.then((deletedEdges) => {
				edges = edges.filter((e) => !deletedEdges.map((edge) => edge.id).includes(e.id));
			});
		nodes = nodes.filter((n) => n.id !== node.id);
		if (selectedNode?.id === node.id) {
			selectedNode = null;
		}
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
		edges = edges.map((e) => { return { ...e, data: { ...e.data, isSelected: false } } });
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
		nodes = nodes.map((n) => { return { ...n, data: { ...n.data, isSelected: false } } });
	};

	const onConnect = async (connection: Connection) => {
		const exists = edges.find(
			(e) => e.source === connection.source && e.target === connection.target
		);
		if (exists) {
			// assert if it is a uuid
			if (exists.id.match(uuidRegex)) {
				console.log('Edge already exists:', exists);
				return;
			} else {
				edges = edges.filter((e) => e.id !== exists.id);
			}
		}

		const newEdge = await edgesApi.connect(boardId, connection.source, connection.target);
		edges = [...edges, { id: newEdge.id, source: connection.source, target: connection.target }];
}	;

	// Keyboard shortcuts
	const onKeyDown = (event: KeyboardEvent) => {
		if (event.key === 'Delete' || event.key === 'Backspace') {
			if (selectedEdge) {
				edgesApi.disconnect(boardId, selectedEdge.id);
				edges = edges.filter((e) => e.id !== selectedEdge?.id);
				selectedEdge = null;
			} else if (selectedNode) {
				deleteNode(selectedNode);
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
		<Realtime {boardId} {userId} {boardUpdate}>
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
					attributionPosition={undefined}
				>
					<Controls />
				</SvelteFlow>
			</div>
		</Realtime>
	</main>
	{#if selectedNode}
		<div
			class="flex flex-col p-4 bg-tertiary border-l border-tertiary-border rounded-l-2xl w-70 h-full justify-between text-tertiary-text z-100!"
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
					if (selectedNode) {
						deleteNode(selectedNode);
					}
				}}
			>
				Delete Node
			</Button>
		</div>
	{/if}

	{#if $isManagingSecrets}
		<SecretsPanel board={boardId} onclose={() => setManagingSecrets(false)} />
	{/if}

	<NodeCreationPanel
		{boardId}
		{creatingNode}
		{paramValues}
		{usableSecrets}
		onclose={() => creatingNode = null}
		onCreateNode={createNode}
	/>
</div>

<style>
	main.dndflow {
		display: flex;
		flex-direction: column;
		/* min-width: 0 lets the flex item shrink below its content, so a wide dock scrolls instead of stretching it */
		flex: 1 1 0;
		min-width: 0;
	}

	:global(.svelte-flow__attribution) {
		display: none;
	}
</style>
