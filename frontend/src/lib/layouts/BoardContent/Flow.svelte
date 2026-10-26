<script lang="ts">
	import { untrack } from 'svelte';
	import {
		SvelteFlow,
		Controls,
		useSvelteFlow,
		type Node,
		type Edge,
		ConnectionMode,
		type Connection
	} from '@xyflow/svelte';

	import { goto } from '$app/navigation';
	import { resolve } from '$app/paths';
	import { page } from '$app/state';

	import { useDnD } from '$providers/DnDProvider.svelte';

	import * as postItsApi from '$services/post-it';
	import * as edgesApi from '$services/edge';
	import type { Board, BoardRole } from '$types/api';
	import SecretsPanel from '$layouts/Secrets/SecretsPanel.svelte';
	import * as secretsApi from '$services/secrets';
	import type { SecretMeta } from '$types/api';
	import { uuid } from '$lib/utils';
	import { nodesMap, parameters } from '$components/Nodes/node-map';
	import { edgesMap } from '$components/Edges/edge-map';
	import { mouses } from '$stores/mouses.svelte';
	import Realtime from './Realtime.svelte';
	import Cursors from './Cursors.svelte';
	import type { NodeMoves, Update } from '$modules/realtime.svelte';
	import { isManagingSecrets, setManagingSecrets } from '$stores/sidebar';
	import { results } from '$stores/postit-results.svelte';
	import { boardPath, nodePath } from '$lib/helpers/routes';
	import NodeDetail, { type PostItData } from './NodeDetail.svelte';

	import NodeConfigurationPanel from '$layouts/NodeConfiguration/NodeConfigurationPanel.svelte';

	let {
		nodes,
		edges,
		role,
		boardId,
		orgId,
		boardUpdate
	}: {
		nodes: Node[];
		edges: Edge[];
		role: BoardRole;
		boardId: string;
		orgId?: string;
		boardUpdate: (update: Update) => void;
	} = $props();

	// Viewers look: no dragging or connecting and no deleting (the board
	// layout hides the cards and credentials; the backend refuses it anyway).
	const canEdit = $derived(role === 'owner' || role === 'editor');

	let settingsVersion = $state(0);

	// The open card lives in the URL (`.../nodes/:nodeId`), so its detail can
	// be shared with a link and the browser's back button closes it.
	const selectedNodeId = $derived(page.params.nodeId ?? null);
	let selectedNode = $derived(nodes.find((n) => n.id === selectedNodeId) ?? null);
	let selectedEdge: Edge | null = $state(null);

	const navigation = { noScroll: true, keepFocus: true };

	const openNode = (id: string) => goto(resolve(nodePath(boardId, id, orgId)), navigation);

	const closeNode = () => {
		if (selectedNodeId) goto(resolve(boardPath(boardId, orgId)), navigation);
	};

	// Everything this user may bind on this board, wherever it lives.
	let usableSecrets = $state<SecretMeta[]>([]);

	// Refreshed whenever the panel closes, so a credential added there is
	// immediately pickable when creating a node.
	$effect(() => {
		if ($isManagingSecrets || !canEdit) return;
		secretsApi
			.usable(boardId)
			.then((s) => (usableSecrets = s))
			.catch(() => (usableSecrets = []));
	});
	let creatingNode = $state<Board['postits'][number] | null>(null);
	let paramValues = $state<Record<string, string>>({});

	let editingNode = $state<PostItData | null>(null);

	const { screenToFlowPosition } = useSvelteFlow();

	mouses.updateMappers(screenToFlowPosition);

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

		if (!type.current || !canEdit) {
			return;
		}

		const position = screenToFlowPosition({
			x: event.clientX,
			y: event.clientY
		});

		// if ((parameters[type.current] ?? []).length === 0) {
		// 	const newPostIt = await postItsApi.create_well_known(boardId, title, type.current, {});
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
			position
		};
	};

	const onBoardClick = () => {
		closeNode();
		selectedEdge = null;

		edges = edges.map((e) => {
			return { ...e, data: { ...e.data, isSelected: false } };
		});
	};

	const persistNode = (action: 'create' | 'update', newNode: Node) => {
		if (!canEdit) return;
		if (action === 'update') {
			nodes = nodes.map((n) => (n.id === newNode.id ? { ...newNode, data: { ...n.data, ...newNode.data } } : n));
			selectedNode = newNode;
			settingsVersion++;
			results.refresh(newNode.id)
		} else if (action === 'create') {
			nodes = [...nodes, { ...newNode }];
		}
		creatingNode = null;
		editingNode = null;
		paramValues = {};
	};

	const deleteNode = async (node: Node) => {
		if (!canEdit) return;
		await postItsApi.del(node.id).then((deletedEdges) => {
			edges = edges.filter((e) => !deletedEdges.map((edge) => edge.id).includes(e.id));
		});
		nodes = nodes.filter((n) => n.id !== node.id);
		results.drop(node.id);
	};

	// Node handlers
	const onNodeClick = async (event: { event: MouseEvent | TouchEvent; node: Node }) => {
		event.event.preventDefault();
		event.event.stopPropagation();
		if (selectedNodeId === event.node.id) {
			closeNode();
		} else {
			openNode(event.node.id);
		}
		selectedEdge = null;
		edges = edges.map((e) => {
			return { ...e, data: { ...e.data, isSelected: false } };
		});
	};

	let realtime = $state<ReturnType<typeof Realtime>>();

	// Others see the cards move while this user drags them; the drop saves.
	const onNodeDrag = (event: { nodes: Node[] }) => {
		realtime?.moveNodes(event.nodes.map((n) => [n.id, n.position.x, n.position.y]));
	};

	// Cards another user is dragging. One this user is dragging stays put.
	const onNodeMoves = (moves: NodeMoves) => {
		const moved = new Map(moves.map(([id, x, y]) => [id, { x, y }]));
		nodes = nodes.map((n) => {
			const position = moved.get(n.id);
			return position && !n.dragging ? { ...n, position } : n;
		});
	};

	const onNodeDragStop = async (event: {
		targetNode: Node | null;
		nodes: Node[];
		event: MouseEvent | TouchEvent;
	}) => {
		const node = event.targetNode;
		if (!node) return;
		await postItsApi.move(node.id, node.position.x, node.position.y);
	};

	// Edge handlers
	const onEdgeClick = async (event: { event: MouseEvent; edge: Edge }) => {
		event.event.preventDefault();
		event.event.stopPropagation();
		if (selectedEdge?.target === event.edge.target && selectedEdge?.source === event.edge.source) {
			selectedEdge = null;
			edges = edges.map((e) => {
				return { ...e, data: { ...e.data, isSelected: false } };
			});
		} else {
			selectedEdge = event.edge;
			edges = edges.map((e) => {
				return { ...e, data: { ...e.data, isSelected: e.id === event.edge.id } };
			});
		}
		closeNode();
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
	};

	// Backspace in a field of the detail drawer edits the field, not the board.
	const isTyping = (target: EventTarget | null) =>
		target instanceof HTMLElement &&
		(target.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName));

	// Keyboard shortcuts. Svelte Flow's own delete key is off (`deleteKey`):
	// it would remove cards and strands locally, for viewers too, without
	// telling the backend.
	const onKeyDown = (event: KeyboardEvent) => {
		if (event.key === 'Escape') {
			closeNode();
			return;
		}
		if (!canEdit || isTyping(event.target)) return;
		if (event.key === 'Delete' || event.key === 'Backspace') {
			if (selectedEdge) {
				edgesApi.disconnect(boardId, selectedEdge.id);
				edges = edges.filter((e) => e.id !== selectedEdge?.id);
				selectedEdge = null;
			} else if (selectedNode) {
				deleteNode(selectedNode);
			}
		}
	};

	const onEditStart = (node: PostItData) => {
		if (!canEdit) return;
		editingNode = { ...node };
		paramValues = node.paramValues;
	};

	// Effects
	$effect(() => {
		window.addEventListener('keydown', onKeyDown);
		return () => window.removeEventListener('keydown', onKeyDown);
	});

	// Highlights the open card, also after a live update replaced the nodes.
	$effect(() => {
		const id = selectedNodeId;
		if (nodes.some((n) => (n.data?.isSelected ?? false) !== (n.id === id))) {
			nodes = nodes.map((n) => ({ ...n, data: { ...n.data, isSelected: n.id === id } }));
		}
	});

	// A link to a card that is not (or no longer) on the board opens the board.
	$effect(() => {
		if (selectedNodeId && !selectedNode) {
			goto(resolve(boardPath(boardId, orgId)), { ...navigation, replaceState: true });
		}
	});

	$effect(() => {
		if (creatingNode) {
			selectedEdge = null;
			untrack(closeNode);
		}
	});
</script>

<div class="relative flex flex-row flex-1 min-h-0 h-full w-full">
	<main class="dndflow">
		<Realtime bind:this={realtime} {boardId} {boardUpdate} {onNodeMoves}>
			<div class="reactflow-wrapper">
				<SvelteFlow
					bind:nodes
					bind:edges
					nodeTypes={nodesMap}
					edgeTypes={edgesMap}
					defaultEdgeOptions={{ type: 'floating' }}
					fitView
					connectionMode={ConnectionMode.Loose}
					nodesDraggable={canEdit}
					nodesConnectable={canEdit}
					deleteKey={null}
					ondragover={onDragOver}
					ondrop={onDrop}
					onnodeclick={onNodeClick}
					onnodedrag={onNodeDrag}
					onnodedragstop={onNodeDragStop}
					onedgeclick={onEdgeClick}
					onconnect={onConnect}
					onpaneclick={onBoardClick}
					colorMode="system"
					class="bg-transparent!"
					attributionPosition={undefined}
				>
					<Controls />
					<Cursors />
				</SvelteFlow>
			</div>
		</Realtime>
	</main>
	{#if selectedNode}
		<NodeDetail node={selectedNode} {canEdit} {settingsVersion} onedit={onEditStart} onback={closeNode} ondelete={deleteNode} />
	{/if}

	{#if $isManagingSecrets}
		<SecretsPanel board={boardId} onclose={() => setManagingSecrets(false)} />
	{/if}

	<NodeConfigurationPanel
		{boardId}
		{creatingNode}
		currentPostIt={editingNode}
		{paramValues}
		{usableSecrets}
		onclose={() => {
			creatingNode = null;
			editingNode = null;
		}}
		onPersistNode={persistNode}
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

	/* The strand being drawn goes under the cards (it renders before them),
	   like the finished floating edges, instead of over its own card. */
	:global(.svelte-flow svg.svelte-flow__connectionline) {
		z-index: 0;
	}
</style>
