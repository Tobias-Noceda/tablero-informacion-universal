<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		{#if data.summary}
			<h3>{data.summary}</h3>
			<h3 class="text-xs opacity-70">{data.start}</h3>
		{:else}
			<h3 class="text-xs opacity-70">No upcoming events</h3>
		{/if}
	{/snippet}
</NodeWrapper>
