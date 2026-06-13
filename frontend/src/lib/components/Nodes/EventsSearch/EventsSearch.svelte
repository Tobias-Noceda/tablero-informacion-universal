<script lang="ts">
  import { Handle, Position, type NodeProps } from '@xyflow/svelte';
  import * as postitAPI from '$services/post-it'
 
  let { id }: NodeProps = $props();

  const request = $derived(postitAPI.execute(id))
</script>

{#await request then data}
  <Handle type="target" position={Position.Top} isConnectable />
  <Handle type="target" position={Position.Bottom} isConnectable />
  <Handle type="target" position={Position.Left} isConnectable />
  <Handle type="target" position={Position.Right} isConnectable />
  <h3>Show: {data.name}</h3>
  <p>Apertura: {data.name}</p>
  <Handle type="source" position={Position.Top} isConnectable />
  <Handle type="source" position={Position.Bottom} isConnectable />
  <Handle type="source" position={Position.Left} isConnectable />
  <Handle type="source" position={Position.Right} isConnectable />
{:catch}
  Upsi
{/await}
