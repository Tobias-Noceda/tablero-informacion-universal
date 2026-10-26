<script lang="ts">
	import { ViewportPortal, useViewport } from '@xyflow/svelte';

	import Cursor from '$components/Cursor/Cursor.svelte';
	import { mouses } from '$stores/mouses.svelte';

	// Inside the viewport the other users' cursors are in flow coordinates,
	// so they stay over the same spot of the board whatever each user's
	// layout, pan or zoom.
	const viewport = useViewport();
</script>

<ViewportPortal target="front">
	{#each mouses.data() as [id, { position, color }] (id)}
		{#if position}
			<Cursor {position} {color} zoom={viewport.current.zoom} />
		{/if}
	{/each}
</ViewportPortal>
