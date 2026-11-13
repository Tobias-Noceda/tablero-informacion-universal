<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3>{data.value} {data.code}</h3>
		<h3 class="text-xs opacity-70">{data.updated}</h3>
	{/snippet}
</NodeWrapper>
