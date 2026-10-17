<script lang="ts">
	import Landing from '$layouts/Landing/Landing.svelte';

	let { data } = $props();
</script>

<Landing org={data.org} />
