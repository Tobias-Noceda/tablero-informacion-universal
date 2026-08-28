<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3>{data.name}</h3>
		<h3>★ {data.stars} · {data.forks} forks · {data.issues} issues</h3>
	{/snippet}
</NodeWrapper>
