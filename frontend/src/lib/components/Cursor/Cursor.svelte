<script lang="ts">
	import type { Position } from '$stores/mouses.svelte';
	import { Spring } from 'svelte/motion';

	interface Params {
		position: Position;
		color: string;
	}

	const { position, color }: Params = $props();

	// svelte-ignore state_referenced_locally
	// Only used for initial position
	const coords = new Spring(position, {
		stiffness: 0.12,
		damping: 0.28
	});

	$effect(() => {
		coords.target = position;
	});

	let width = $state(0);
	let height = $state(0);

	const x = $derived(coords.current.x);
	const y = $derived(coords.current.y);

	function visible() {
		const margin = 10;
		return -margin < x && x < width - margin && -margin < y && y < height - margin;
	}
</script>

<svelte:window bind:innerWidth={width} bind:innerHeight={height} />

{#if visible()}
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
		class="size-2 absolute"
		style:background-color={color}
		style:top="{y}px"
		style:left="{x}px"
	></div>
{/if}
