<script lang="ts">
    import { Handle, Position, type NodeProps } from '@xyflow/svelte';
    import * as postitAPI from '$services/post-it';
    import type { Snippet } from 'svelte';

    let { id, children }: NodeProps & { children: Snippet<[Record<string, string>]> } = $props();

    const request = $derived(postitAPI.execute(id));
</script>

<div class="bg-main p-4 rounded-lg">
    <Handle type="source" id={`top-${id}`} position={Position.Top} class="handle" isConnectable />
    <Handle type="source" id={`right-${id}`} position={Position.Right} class="handle" isConnectable />
    {#await request then data}
        {@render children(data)}
    {:catch}
        An error has occurred
    {/await}
    <Handle type="source" id={`bottom-${id}`} position={Position.Bottom} class="handle" isConnectable />
    <Handle type="source" id={`left-${id}`} position={Position.Left} class="handle" isConnectable />
</div>