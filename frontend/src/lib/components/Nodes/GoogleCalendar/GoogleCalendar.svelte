<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';
	import { m } from '$lib/paraglide/messages';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		{#if data.summary}
			<h3>{data.summary}</h3>
			<h3 class="text-xs opacity-70">{data.start}</h3>
		{:else}
			<h3 class="text-xs opacity-70">{m['nodes.google_calendar_empty']()}</h3>
		{/if}
	{/snippet}
</NodeWrapper>
