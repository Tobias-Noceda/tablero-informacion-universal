<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3>Riesgo país {data.valor}</h3>
		<h3>{data.fecha}</h3>
	{/snippet}
</NodeWrapper>
