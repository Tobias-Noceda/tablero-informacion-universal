<script lang="ts">
	import { untrack, type Snippet } from 'svelte';

	import * as realtime from '$modules/realtime.svelte';
	import { session } from '$modules/session.svelte';
	import { mouses, type Position } from '$stores/mouses.svelte';

	type Props = {
		children: Snippet;
		boardId: string;
		boardUpdate: (update: realtime.Update) => void;
		// Cards another user is dragging, where they are now.
		onNodeMoves: (moves: realtime.NodeMoves) => void;
	};

	let { children, boardId, boardUpdate, onNodeMoves }: Props = $props();

	let frame: number | null = null;
	let nodesFrame: number | null = null;
	// Null when the board has no live updates (the service refused the socket).
	let connection: Promise<realtime.Connection | null>;

	// https://github.com/sveltejs/svelte/issues/13249#issuecomment-2351801858
	$effect.pre(() => {
		const board = boardId;
		// Only the board reopens the connection: renewing the session replaces
		// `session.user`, and the socket renews its own token.
		connection = untrack(() =>
			realtime.connect(board, session.user!, boardUpdate, (moves) => onNodeMoves(moves))
		).catch((err) => {
			console.warn('Live updates are off for this board:', err);
			return null;
		});

		return async () => {
			if (frame) cancelAnimationFrame(frame);
			if (nodesFrame) cancelAnimationFrame(nodesFrame);
			(await connection)?.close();
		};
	});

	// The latest pointer position, sent once per frame. Converted in the frame,
	// so a wheel zoom has already moved the viewport.
	function move(e: PointerEvent | DragEvent | WheelEvent) {
		const client: Position = { x: e.clientX, y: e.clientY };
		if (frame !== null) cancelAnimationFrame(frame);
		frame = requestAnimationFrame(() => {
			frame = null;
			const { x, y } = mouses.convert(client);
			connection.then((r) => r?.update([x, y]));
		});
	}

	let pendingMoves: realtime.NodeMoves = [];

	// Shows the cards this user drags to everyone else, at most once per frame.
	export function moveNodes(moves: realtime.NodeMoves) {
		pendingMoves = moves;
		if (nodesFrame !== null) return;
		nodesFrame = requestAnimationFrame(() => {
			nodesFrame = null;
			const moves = pendingMoves;
			connection.then((r) => r?.moveNodes(moves));
		});
	}
</script>

<!-- d3 (dragging a card, panning) stops mousemove; pointermove still arrives.
	 Dragging a card in from the dock only fires dragover. -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div onpointermove={move} ondragover={move} onwheel={move} class="contents">
	{@render children()}
</div>
