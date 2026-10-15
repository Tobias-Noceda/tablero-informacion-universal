<script lang="ts">
	import type { Snippet } from 'svelte';

	import BoardContent from '$layouts/BoardContent/BoardContent.svelte';
	import type { BoardData } from '$layouts/BoardContent/type';

	// A layout, so opening a card (`nodes/[nodeId]`) keeps the board mounted.
	let { data, children }: { data: BoardData; children: Snippet } = $props();
</script>

<BoardContent {data} />
{@render children()}
