<script lang="ts">
  import { type NodeProps } from '@xyflow/svelte';
	import NodeWrapper from '../NodeWrapper.svelte';
 
  let props: NodeProps = $props();
</script>

<NodeWrapper {...props}>
  {#snippet children(data)}
    <h3>Máxima: {data.max}</h3>
    <p>Mínima: {data.min}</p>
  {/snippet}
</NodeWrapper>
