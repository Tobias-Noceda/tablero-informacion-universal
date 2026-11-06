<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import { getLocale } from '$lib/paraglide/runtime';
	import { display } from '$lib/title';
	import { wellKnowns } from '$stores/well-knowns.svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();

	// A well-known without a hand-made card: each output, labelled. A card
	// whose custom type was deleted has no labels left, so it shows what it
	// answered as is.
	const known = $derived(!!wellKnowns.byKey(props.type));
	const outputs = $derived(wellKnowns.outputs(props.type));
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3 class="text-lg font-bold">{wellKnowns.name(props.type)}</h3>
		{#if known}
			{#each outputs as output (output.key)}
				<p>{output[`label_${getLocale()}`]}: {display(data[output.key])}</p>
			{/each}
		{:else}
			{#each Object.entries(data ?? {}) as [key, value] (key)}
				<p>{key}: {display(value)}</p>
			{/each}
		{/if}
	{/snippet}
</NodeWrapper>
