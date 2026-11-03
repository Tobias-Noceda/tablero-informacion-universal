<script lang="ts">
	import { type NodeProps } from '@xyflow/svelte';
	import { getLocale } from '$lib/paraglide/runtime';
	import { wellKnowns } from '$stores/well-knowns.svelte';
	import NodeWrapper from '../NodeWrapper.svelte';

	let props: NodeProps = $props();

	// A well-known without a hand-made card: each output, labelled.
	const outputs = $derived(wellKnowns.outputs(props.type));
</script>

<NodeWrapper {...props}>
	{#snippet children(data)}
		<h3 class="text-lg font-bold">{wellKnowns.name(props.type)}</h3>
		{#each outputs as output (output.key)}
			<p>{output[`label_${getLocale()}`]}: {data[output.key] ?? ''}</p>
		{/each}
	{/snippet}
</NodeWrapper>
