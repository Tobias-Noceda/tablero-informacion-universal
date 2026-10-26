<script lang="ts">
	import Icon from '$components/Icon/Icon.svelte';
	import type { Position } from '$stores/mouses.svelte';
	import { Spring } from 'svelte/motion';

	interface Params {
		// Flow coordinates: drawn inside the viewport, which pans and zooms it.
		position: Position;
		color: string;
		// The viewport's zoom, undone so the cursor keeps its size.
		zoom?: number;
	}

	const { position, color, zoom = 1 }: Params = $props();

	// svelte-ignore state_referenced_locally
	// Only used for initial position
	const coords = new Spring(position, {
		stiffness: 0.12,
		damping: 0.28
	});

	$effect(() => {
		coords.target = position;
	});

	const x = $derived(coords.current.x);
	const y = $derived(coords.current.y);
</script>

<!-- <svg
	xmlns="http://www.w3.org/2000/svg"
	class="w-6 h-8 absolute fill-none"
	viewBox="0 0 24 36"
	// style:top={y}
	// style:left={x}
	style:transform="translateX(${x}px) translateY(${y}px)"
>
	<path
		d="M5.65376 12.3673H5.46026L5.31717 12.4976L0.500002 16.8829L0.500002 1.19841L11.7841 12.3673H5.65376Z"
		fill={color}
	/>
</svg> -->
<div
	class="pointer-events-none absolute top-0 left-0 size-2 origin-top-left"
	style:transform="translate({x}px, {y}px) scale({1 / zoom})"
	style:color={color}
>
	<Icon name="cursor" class="w-6 h-8" />
</div>
