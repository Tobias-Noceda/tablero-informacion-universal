<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		{@const change = Number(data.change)}
		<h3>{data.price}</h3>
		<h3>{change > 0 ? '+' : ''}{change.toFixed(2)}% 24h</h3>
	{/snippet}
</NodeWrapper>
