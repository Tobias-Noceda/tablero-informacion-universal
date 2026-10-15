<script lang="ts">
	import BoardContent from '$layouts/BoardContent/BoardContent.svelte';
	import type { BoardData } from '$layouts/BoardContent/type';

	let { data }: { data: BoardData } = $props();
</script>

<BoardContent {data} />
