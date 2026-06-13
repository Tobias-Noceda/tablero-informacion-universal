<script lang="ts">
	import { Handle, Position, type NodeProps } from '@xyflow/svelte';
	import * as postitAPI from '$services/post-it';

	let { id }: NodeProps = $props();

	const request = $derived(postitAPI.execute(id));
</script>

<div class="bg-main p-4 rounded-lg">
	<Handle type="target" position={Position.Bottom} isConnectable />
	{#await request then data}
		<p>{data.text}</p>
	{:catch}
		An error has occurred
	{/await}
	<Handle type="source" position={Position.Top} isConnectable />
</div>
