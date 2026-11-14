<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import { renderTitle } from '$lib/title';
	import { cn } from '$lib/utils';
	import NodeWrapper from '../NodeWrapper.svelte';
	import ImageView from './ImageView.svelte';

	let props: NodeProps = $props();

	const title = $derived(props.data?.title as { text: string; vars: boolean } | undefined);
</script>

<!-- The picture fills the card; the card's title is its caption. -->
<NodeWrapper {...props} media>
	{#snippet children(data)}
		{@const caption = title?.text ? renderTitle(title.text, data) : ''}
		<figure class="flex h-full w-full flex-col gap-2">
			<ImageView
				src={data.url}
				alt={caption}
				linkable={false}
				class={cn(
					'block rounded object-contain',
					props.width ? 'w-full' : 'max-w-80',
					props.height ? 'min-h-0 flex-1' : 'max-h-60'
				)}
			/>
			{#if caption}
				<figcaption class="shrink-0 truncate text-sm font-semibold">{caption}</figcaption>
			{/if}
		</figure>
	{/snippet}
</NodeWrapper>
