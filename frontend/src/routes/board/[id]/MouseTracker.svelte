<script lang="ts">
	import { type Snippet } from 'svelte';

	import Cursor from '$components/Cursor/Cursor.svelte';

	import * as realtime from '$modules/realtime.svelte';
	import { mouses } from '$stores/mouses.svelte';

	type Props = {
		children: Snippet;
		boardId: string;
		boardUpdate: (update: realtime.Update) => void;
	};

	let { children, boardId, boardUpdate }: Props = $props();

	let frame: number | null = null;

	$effect(() => {
		const connections = realtime.connect(boardId, boardUpdate);

		return () => {
			if (frame) cancelAnimationFrame(frame);
			connections.then((off) => off());
		};
	});

	function move(e: MouseEvent) {
		if (frame !== null) {
			return;
		}

		frame = requestAnimationFrame(() => {
			frame = null;
			const { x, y } = mouses.convert({ x: e.clientX, y: e.clientY });
			realtime.update([x, y]);
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
