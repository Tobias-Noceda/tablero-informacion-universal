<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3>Compra {data.compra}</h3>
		<h3>Venta {data.venta}</h3>
	{/snippet}
</NodeWrapper>
