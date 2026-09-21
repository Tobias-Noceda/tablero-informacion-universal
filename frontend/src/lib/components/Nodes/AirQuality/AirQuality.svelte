<script lang="ts">
	import type { NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3>AQI {data.aqi}</h3>
		<h3>PM2.5 {data.pm25}</h3>
	{/snippet}
</NodeWrapper>
