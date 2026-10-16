<script lang="ts">
	import { untrack, type Snippet } from 'svelte';

	import Cursor from '$components/Cursor/Cursor.svelte';

	import * as realtime from '$modules/realtime.svelte';
	import { session } from '$modules/session.svelte';
	import { mouses } from '$stores/mouses.svelte';

	type Props = {
		children: Snippet;
		boardId: string;
		boardUpdate: (update: realtime.Update) => void;
	};

	let { children, boardId, boardUpdate }: Props = $props();

	let frame: number | null = null;
	// Null when the board has no live updates (the service refused the socket).
	let connection: Promise<realtime.Connection | null>;

	// https://github.com/sveltejs/svelte/issues/13249#issuecomment-2351801858
	$effect.pre(() => {
		const board = boardId;
		// Only the board reopens the connection: renewing the session replaces
		// `session.user`, and the socket renews its own token.
		connection = untrack(() => realtime.connect(board, session.user!, boardUpdate)).catch((err) => {
			console.warn('Live updates are off for this board:', err);
			return null;
		});

		return async () => {
			if (frame) cancelAnimationFrame(frame);
			(await connection)?.close();
		};
	});

	function move(e: MouseEvent) {
		if (frame !== null) {
			return;
		}

		frame = requestAnimationFrame(() => {
			frame = null;
			const { x, y } = mouses.convert({ x: e.clientX, y: e.clientY });
			connection.then((r) => r?.update([x, y]));
		});
	}
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div onmousemove={move} class="contents">
	{@render children()}
</div>

{#each mouses.data() as [id, { position, color }] (id)}
	{#if position}
		<Cursor {position} {color} />
	{/if}
{/each}
