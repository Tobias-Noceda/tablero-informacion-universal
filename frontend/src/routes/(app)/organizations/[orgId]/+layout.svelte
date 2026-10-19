<script lang="ts">
	import BoardCreator from '$layouts/BoardCreator/BoardCreator.svelte';

	let { data, children } = $props();
</script>

{@render children()}

<BoardCreator org={data.org} />
