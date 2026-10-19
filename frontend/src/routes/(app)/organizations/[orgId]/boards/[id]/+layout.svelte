<script lang="ts">
	import BoardContent from '$layouts/BoardContent/BoardContent.svelte';

	// A layout, so opening a card (`nodes/[nodeId]`) keeps the board mounted.
	let { data, children } = $props();
</script>

<BoardContent {data} />
{@render children()}
