<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3>Show: {data.name}</h3>
		<p>Apertura: {data.sales}</p>
	{/snippet}
</NodeWrapper>
