<script lang="ts">
    import './index.css';
    import { Handle, Position, type NodeProps } from '@xyflow/svelte';
    import * as postitAPI from '$services/post-it';
    import { renderTitle } from '$lib/title';
    import type { Snippet } from 'svelte';

    let { id, data, children }: NodeProps & { children: Snippet<[Record<string, string>]> } = $props();

    const title = $derived(data?.title as { text: string, vars: boolean } | undefined);
    // A plain title is all the card shows, so it needs no data.
    const plainTitle = $derived(!!title?.text && !title.vars);
    const request = $derived(plainTitle ? null : postitAPI.execute(id));

    const isSelected = $derived.by(() => data?.isSelected ?? false);
</script>

<div
    class="bg-main hover:bg-main-hover p-4 rounded-lg customNode text-main-text"
    style={isSelected ? 'background-color: var(--color-main-hover)' : undefined}
>
    <Handle class="customHandle" position={Position.Left} type="source" />
    {#if plainTitle}
        <h3 class="text-lg font-bold">{title?.text}</h3>
    {:else}
        {#await request then response}
            {#if title?.text}
                <!-- Plain text interpolation: Svelte escapes whatever the values hold. -->
                <h3 class="text-lg font-bold">{renderTitle(title.text, response)}</h3>
            {:else if response}
                {@render children(response)}
            {/if}
        {:catch}
            An error has occurred
        {/await}
    {/if}
</div>
