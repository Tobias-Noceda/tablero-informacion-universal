<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3>{data.title}</h3>
		{#if data.image}
			<img src={data.image} alt={data.title} />
		{/if}
		<p>{data.date}</p>
	{/snippet}
</NodeWrapper>

<style>
	img {
		max-width: 100%;
		border-radius: 4px;
	}
</style>
