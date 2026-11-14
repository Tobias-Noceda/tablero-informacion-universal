<script lang="ts">
	import { m } from '$lib/paraglide/messages';
	import { isLink } from '$lib/helpers/content-rendering';

	let {
		src,
		alt,
		linkable = true,
		class: className
	}: {
		src: unknown;
		alt: string;
		/** On the canvas a click opens the card, so the address is shown, not linked. */
		linkable?: boolean;
		class?: string;
	} = $props();

	// Only http(s) addresses: the value comes from whoever made the card.
	const url = $derived(isLink(src) ? src : null);
	// The address that failed to load; a new one gets its own try.
	let failedUrl = $state<string | null>(null);
	const failed = $derived(url !== null && failedUrl === url);
</script>

{#if !url}
	<p class="text-sm opacity-80">{m['image.missing']()}</p>
{:else if failed}
	<!-- A page or anything else that is not a picture: say so instead of a broken image. -->
	<p class="text-sm break-all">
		<span class="opacity-80">{m['image.not_an_image']()}</span>
		{#if linkable}
			<!-- An external address the card holds, not an app route. -->
			<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
			<a href={url} target="_blank" rel="noopener noreferrer" class="underline">{url}</a>
		{:else}
			{url}
		{/if}
	</p>
{:else}
	<img src={url} {alt} class={className} draggable="false" onerror={() => (failedUrl = url)} />
{/if}
