<script lang="ts">
    import './index.css';
    import { Handle, Position, type NodeProps } from '@xyflow/svelte';
    import { results } from '$stores/postit-results.svelte';
    import { renderTitle } from '$lib/title';
    import type { Snippet } from 'svelte';
	import { isImageLink, isLink } from '$lib/helpers/content-rendering';

    let { id, data, children }: NodeProps & { children: Snippet<[Record<string, string>]> } = $props();

    const title = $derived(data?.title as { text: string, vars: boolean } | undefined);
    // A plain title is all the card shows, so it needs no data.
    const plainTitle = $derived(!!title?.text && !title.vars);
    const request = $derived(plainTitle ? null : results.get(id));

    const isSelected = $derived.by(() => data?.isSelected ?? false);

    function shouldRenderImage(title: string, response: Record<string, string> | null): boolean {
        const renderedTitle = renderTitle(title, response);

        return renderedTitle.split(/\s+/).length === 1 && isLink(renderedTitle) && isImageLink(renderedTitle);
    }
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
                {#if shouldRenderImage(title.text, response)}
                    <!-- An external address the card returned, not an app route. -->
                    <!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
                    <a href={renderTitle(title.text, response)} target="_blank" rel="noopener noreferrer" class="underline">
                        <img src={renderTitle(title.text, response)} alt={renderTitle(title.text, response)} class="max-h-40 max-w-full" />
                    </a>
                {:else}
                    <h3 class="text-lg font-bold">{renderTitle(title.text, response)}</h3>
                {/if}
            {:else if response}
                {@render children(response)}
            {/if}
        {:catch}
            An error has occurred
        {/await}
    {/if}
</div>
