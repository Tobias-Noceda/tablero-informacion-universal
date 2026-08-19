<script lang="ts">
    import './index.css';
    import { Handle, Position, type NodeProps } from '@xyflow/svelte';
    import * as postitAPI from '$services/post-it';
    import type { Snippet } from 'svelte';

    let { id, data, children }: NodeProps & { children: Snippet<[Record<string, string>]> } = $props();

    const request = $derived(postitAPI.execute(id));

    const isSelected = $derived.by(() => data?.isSelected ?? false);
</script>

<div
    class="bg-main hover:bg-main-hover p-4 rounded-lg customNode"
    style={isSelected ? 'background-color: var(--color-main-hover)' : undefined}
>
    <Handle class="customHandle" position={Position.Left} type="source" />
    {#await request then data}
        {@render children(data)}
    {:catch}
        An error has occurred
    {/await}
</div>
