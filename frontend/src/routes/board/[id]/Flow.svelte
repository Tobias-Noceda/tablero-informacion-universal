<script lang="ts">
	import { SvelteFlow, Controls, useSvelteFlow, type Node, type Edge, ConnectionMode, type Connection } from '@xyflow/svelte';

	import { useDnD } from './DnDProvider.svelte';
	import Dock from './Dock.svelte';

	import * as postItsApi from '$services/post-it';
	import * as edgesApi from '$services/edge';
	import type { Board } from '$types/api';
	import Modal from '$components/Modal/Modal.svelte';
	import Input from '$components/Input/Input.svelte';
	import Button from '$components/Button/Button.svelte';
	import SecretsPanel from '$components/Secrets/SecretsPanel.svelte';
	import * as secretsApi from '$services/secrets';
	import { CURRENT_USER } from '$modules/api.svelte';
	import { groupByOrigin } from '$lib/secrets/origin';
	import { bindingsFor, refKey } from '$lib/secrets/binding';
	import type { SecretMeta } from '$types/api';
	import { m } from '$lib/paraglide/messages';
	import { uuid } from '$lib/utils';
	import { nodesMap, parameters } from '$components/Nodes/node-map';
	import { edgesMap } from '$components/Edges/edge-map';
	import { mouses } from '$stores/mouses.svelte';
	import Realtime from './Realtime.svelte';
	import type { Update } from '$modules/sockets.svelte';

	let { nodes, edges, name, boardId, userId, boardUpdate }: {
		nodes: Node[],
		edges: Edge[],
		name: string,
		boardId: string,
		userId: string,
		boardUpdate: (update: Update) => void
	} = $props();

	let selectedNode: Node | null = $state(null);
	let selectedEdge: Edge | null = $state(null);

	let managingSecrets = $state(false);
	// Everything this user may bind on this board, wherever it lives.
	let usableSecrets = $state<SecretMeta[]>([]);
	const pickable = $derived(groupByOrigin(usableSecrets, boardId, CURRENT_USER));
	const byKey = $derived(new Map(usableSecrets.map((s) => [refKey(s), s])));

	// Refreshed whenever the panel closes, so a credential added there is
	// immediately pickable when creating a node.
	$effect(() => {
		if (managingSecrets) return;
		secretsApi.usable(boardId).then((s) => (usableSecrets = s)).catch(() => (usableSecrets = []));
	});
	let creatingNode = $state<Board['postits'][number] | null>(null);
	let paramValues = $state<Record<string, string>>({});
	// A secret parameter holds the picked secret's key, not its name.
	let pickedKeys = $state<Record<string, string>>({});

	const creatingParams = $derived(creatingNode ? (parameters[creatingNode.type!] ?? []) : []);
	const missingRequired = $derived(
		creatingParams.some((p) => {
			const value = p.type === 'secret' ? pickedKeys[p.key] : paramValues[p.key];
			return p.default === undefined && !(value ?? '').trim();
		})
	);

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
			id: uuid(),
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

		const picked = Object.fromEntries(
			creatingParams
				.filter((p) => p.type === 'secret' && byKey.has(pickedKeys[p.key]))
				.map((p) => [p.key, byKey.get(pickedKeys[p.key])!])
		);
		const secrets = bindingsFor(picked, boardId);

		const params = {
			...Object.fromEntries(
				creatingParams
					.filter((p) => p.type !== 'secret')
					.map((p) => [p.key, (paramValues[p.key] ?? '').trim()])
			),
			...secrets.params
		};

		const newPostIt = await postItsApi.create_well_known(boardId, creatingNode.type!, params, secrets.bindings);
		await postItsApi.move(newPostIt.id, creatingNode.position.x, creatingNode.position.y);

		nodes = [...nodes, { ...creatingNode, id: newPostIt.id, data: params }];
		creatingNode = null;
		paramValues = {};
		pickedKeys = {};
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
					<div class="absolute top-0 inset-x-0 flex flex-row items-center justify-between p-3 z-100! bg-transparent pointer-events-none">
						<h1 class="text-2xl font-bold">{name}</h1>
						<Button class="pointer-events-auto" variant="secondary" onclick={() => (managingSecrets = true)}>
							{m['secrets.title']()}
						</Button>
					</div>
				</SvelteFlow>
			</div>
		</Realtime>
		<Dock />
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

	{#if managingSecrets}
		<SecretsPanel board={boardId} onclose={() => (managingSecrets = false)} />
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
				{#if param.type === 'secret'}
					<label class="flex flex-col gap-1 text-sm">
						{param.label}
						{#if pickable.length === 0}
							<span class="text-xs text-destructive">{m['secrets.no_credentials']()}</span>
						{:else}
							<select
								class="bg-background border border-main-border rounded-md px-2 py-1"
								bind:value={pickedKeys[param.key]}
							>
								<option value="">{m['secrets.pick_credential']()}</option>
								{#each pickable as group (group.origin)}
									<optgroup label={m[`secrets.origin_${group.origin}`]()}>
										{#each group.secrets as secret (refKey(secret))}
											<option value={refKey(secret)}>{secret.name} ({secret.provider ?? secret.kind})</option>
										{/each}
									</optgroup>
								{/each}
							</select>
						{/if}
					</label>
				{:else}
					<Input
						label={param.label}
						placeholder={param.placeholder}
						type={param.type === 'number' ? 'number' : 'text'}
						required={param.default === undefined}
						bind:value={paramValues[param.key]}
					/>
				{/if}
			{/each}
		</Modal>
	{/if}
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
