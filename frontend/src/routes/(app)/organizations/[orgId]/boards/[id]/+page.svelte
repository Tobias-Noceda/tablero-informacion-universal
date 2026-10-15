<script lang="ts">
	import BoardContent from '$layouts/BoardContent/BoardContent.svelte';

	let { data } = $props();
</script>

<BoardContent {data} />
