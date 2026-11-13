<script lang="ts">
    /* eslint-disable svelte/no-navigation-without-resolve */
	import './index.css';
	import { Handle, NodeResizer, NodeToolbar, Position, type NodeProps } from '@xyflow/svelte';
	import { results } from '$stores/postit-results.svelte';
	import { renderTitle } from '$lib/title';
	import type { Snippet } from 'svelte';
	import { isImageLink, isLink } from '$lib/helpers/content-rendering';
	import { cn } from '$lib/utils';
	import { getCardEditor } from '$lib/cards/editor';
	import {
		cardBorder,
		cardFill,
		cardText,
		MAX_CARD_HEIGHT,
		MAX_CARD_WIDTH,
		MIN_CARD_HEIGHT,
		MIN_CARD_WIDTH,
		type CardLook
	} from '$lib/cards/style';
	import CardToolbar from './CardToolbar.svelte';

	let {
		id,
		data,
		width,
		height,
		media = false,
		children
	}: NodeProps & {
		children: Snippet<[Record<string, string>]>;
		/** A picture card: always shows its content, which fills the card, instead of its title. */
		media?: boolean;
	} = $props();

	const editor = getCardEditor();

	const title = $derived(data?.title as { text: string; vars: boolean } | undefined);
	// A plain title is all the card shows, so it needs no data.
	const plainTitle = $derived(!media && !!title?.text && !title.vars);
	const request = $derived(plainTitle ? null : results.get(id));
	// Fetched again now and then while the card shows its data.
	$effect(() => {
		if (!plainTitle) return results.watch(id);
	});

	const isSelected = $derived.by(() => data?.isSelected ?? false);
	// An editor restyles and resizes the open card.
	const showTools = $derived((editor?.canEdit ?? false) && isSelected === true);

	// A side without a size of its own follows the content, up to a limit.
	const look = $derived((data?.look ?? {}) as CardLook);
	const sized = $derived(!!width || !!height);
	// Where a resize started, to tell whether it also moved the card.
	let resizeStart: { x: number; y: number } | null = null;

	function shouldRenderImage(title: string, response: Record<string, string> | null): boolean {
		const renderedTitle = renderTitle(title, response);

		return (
			renderedTitle.split(/\s+/).length === 1 && isLink(renderedTitle) && isImageLink(renderedTitle)
		);
	}
</script>

<NodeToolbar isVisible={showTools} position={Position.Top}>
	<CardToolbar {look} {sized} onchange={(patch) => editor?.restyle(id, patch)} />
</NodeToolbar>

<NodeResizer
	isVisible={showTools}
	minWidth={MIN_CARD_WIDTH}
	minHeight={MIN_CARD_HEIGHT}
	maxWidth={MAX_CARD_WIDTH}
	maxHeight={MAX_CARD_HEIGHT}
	color="var(--app-foreground)"
	onResizeStart={(_, bounds) => (resizeStart = bounds)}
	onResizeEnd={(_, bounds) =>
		editor?.resized(id, bounds, !!resizeStart && (resizeStart.x !== bounds.x || resizeStart.y !== bounds.y))}
/>

<div
	class={cn(
		'bg-main hover:bg-main-hover rounded-lg customNode text-main-text overflow-hidden',
		media ? 'p-2' : 'p-4',
		isSelected && 'bg-main-hover',
		// A fill of its own replaces the hover colour; white can only darken.
		look.color === 'white'
			? isSelected ? 'brightness-95' : 'hover:brightness-95'
			: look.color && (isSelected ? 'brightness-110' : 'hover:brightness-110'),
		width && 'w-full',
		height ? 'h-full' : !media && 'max-h-34!'
	)}
	style:background-color={cardFill(look.color)}
	style:border={cardBorder(look)}
	style:color={cardText(look.color)}
>
	<Handle class="customHandle" position={Position.Left} type="source" />
	<div
		class={cn(
			'overflow-hidden text-ellipsis whitespace-wrap',
			!width && !media && 'max-w-92!',
			height ? 'h-full' : !media && 'max-h-26! line-clamp-3'
		)}
	>
		{#if plainTitle}
			<h3 class="text-lg font-bold">{title?.text}</h3>
		{:else}
			{#await request then response}
				{#if media}
					{#if response}{@render children(response)}{/if}
				{:else if title?.text}
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
