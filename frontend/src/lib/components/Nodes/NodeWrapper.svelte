<script lang="ts">
    /* eslint-disable svelte/no-navigation-without-resolve */
	import './index.css';
	import { Handle, Position, type NodeProps } from '@xyflow/svelte';
	import { results } from '$stores/postit-results.svelte';
	import { renderTitle } from '$lib/title';
	import type { Snippet } from 'svelte';
	import { isImageLink, isLink } from '$lib/helpers/content-rendering';
	import { cn } from '$lib/utils';

	let { id, data, children }: NodeProps & { children: Snippet<[Record<string, string>]> } =
		$props();

	const title = $derived(data?.title as { text: string; vars: boolean } | undefined);
	// A plain title is all the card shows, so it needs no data.
	const plainTitle = $derived(!!title?.text && !title.vars);
	const request = $derived(plainTitle ? null : results.get(id));

	const isSelected = $derived.by(() => data?.isSelected ?? false);

	function shouldRenderImage(title: string, response: Record<string, string> | null): boolean {
		const renderedTitle = renderTitle(title, response);

		return (
			renderedTitle.split(/\s+/).length === 1 && isLink(renderedTitle) && isImageLink(renderedTitle)
		);
	}
</script>

<div
	class={cn(
		'bg-main hover:bg-main-hover p-4 rounded-lg customNode text-main-text',
		isSelected && 'bg-main-hover',
        'max-h-34! overflow-hidden'
	)}
>
	<Handle class="customHandle" position={Position.Left} type="source" />
	<div
		class={cn(
			'max-w-92! max-h-26! overflow-hidden',
			'overflow-hidden text-ellipsis whitespace-wrap line-clamp-3'
		)}
	>
		{#if plainTitle}
			<h3 class="text-lg font-bold">{title?.text}</h3>
		{:else}
			{#await request then response}
				{#if title?.text}
					<!-- Plain text interpolation: Svelte escapes whatever the values hold. -->
					{#if shouldRenderImage(title.text, response)}
						<!-- An external address the card returned, not an app route. -->
						<a
							href={renderTitle(title.text, response)}
							target="_blank"
							rel="noopener noreferrer"
							class="underline"
						>
							<img
								src={renderTitle(title.text, response)}
								alt={renderTitle(title.text, response)}
								class="max-h-40 max-w-full"
							/>
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
</div>
