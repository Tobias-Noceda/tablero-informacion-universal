<script lang="ts">
	import ResponseTree from './ResponseTree.svelte';
	import { childrenOf, leafText, type JsonPath } from '$lib/cards/preview';
	import { m } from '$lib/paraglide/messages';

	interface Props {
		value: unknown;
		/** Where `value` sits in the whole response. */
		path?: JsonPath;
		/** A value picked: its path, and whether every element of its list was asked for. */
		onpick: (path: JsonPath, every: boolean) => void;
		/** Inside a list: a leaf can also be picked for every element. */
		inList?: boolean;
		/** Shown open at first: the first levels. */
		depth?: number;
	}

	let { value, path = [], onpick, inList = false, depth = 0 }: Props = $props();

	const branch = $derived(value !== null && typeof value === 'object');
	const children = $derived(childrenOf(value));
	const isList = $derived(Array.isArray(value));
	const pickClass =
		'cursor-pointer rounded px-1 text-left font-mono hover:bg-main-hover focus:outline-none focus:ring-2 focus:ring-main-border';
</script>

{#if branch}
	<ul class="flex flex-col border-l border-main-border pl-3">
		{#each children.entries as [key, child] (key)}
			<li class="flex flex-col">
				{#if child !== null && typeof child === 'object'}
					<details open={depth < 1}>
						<summary class="cursor-pointer font-mono text-xs opacity-80">
							{typeof key === 'number' ? `[${key}]` : key}
							<span class="opacity-60">
								{Array.isArray(child) ? m['card_preview.items']({ count: child.length }) : '{…}'}
							</span>
						</summary>
						<ResponseTree
							value={child}
							path={[...path, key]}
							{onpick}
							inList={inList || isList}
							depth={depth + 1}
						/>
					</details>
				{:else}
					<div class="flex flex-wrap items-baseline gap-1 text-xs">
						<span class="font-mono opacity-80">{typeof key === 'number' ? `[${key}]` : key}:</span>
						<button
							type="button"
							class={pickClass}
							title={m['card_preview.pick']()}
							onclick={() => onpick([...path, key], false)}
						>
							{leafText(child)}
						</button>
						{#if inList || isList}
							<button
								type="button"
								class="cursor-pointer rounded border border-main-border px-1 text-[10px] hover:bg-main-hover"
								title={m['card_preview.pick_all_hint']()}
								onclick={() => onpick([...path, key], true)}
							>
								{m['card_preview.pick_all']()}
							</button>
						{/if}
					</div>
				{/if}
			</li>
		{/each}
		{#if children.hidden > 0}
			<li class="text-xs opacity-60">{m['card_preview.more']({ count: children.hidden })}</li>
		{/if}
	</ul>
{:else}
	<button type="button" class={pickClass + ' text-xs'} onclick={() => onpick(path, false)}>
		{leafText(value)}
	</button>
{/if}
