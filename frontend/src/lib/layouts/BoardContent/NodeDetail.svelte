<script lang="ts">
	import { isHttpError } from '@sveltejs/kit';
	import type { Node } from '@xyflow/svelte';

	import Button from '$components/Button/Button.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import { parameters } from '$components/Nodes/node-map';
	import { m } from '$lib/paraglide/messages';
	import { display, renderTitle } from '$lib/title';
	import * as postItsApi from '$services/post-it';
	import { results } from '$stores/postit-results.svelte';
	import { cn } from '$lib/utils';
	import { capitalize, isImageLink, isLink } from '$lib/helpers/content-rendering';

	let {
		node,
		canEdit,
		onback,
		ondelete
	}: {
		node: Node;
		canEdit: boolean;
		onback: () => void;
		ondelete: (node: Node) => void;
	} = $props();

	const title = $derived(node.data?.title as { text: string; vars: boolean } | undefined);

	// What kind of card it is, as the cards sidebar names it.
	const kind = $derived.by(() => {
		const key = `nodes.${node.type}`;
		const messages = m as unknown as Record<string, (() => string) | undefined>;
		return messages[key]?.() ?? node.type ?? '';
	});

	const scrollClass = cn(
		// Custom scrollbar styles
		'scrollbar-thin scrollbar-thumb-main scrollbar-track-transparent',
		'[&::-webkit-scrollbar]:w-2',
		'[&::-webkit-scrollbar-track]:bg-transparent',
		'[&::-webkit-scrollbar-thumb]:bg-main',
		'[&::-webkit-scrollbar-thumb]:rounded-full',
		'[&::-webkit-scrollbar-thumb]:hover:bg-main-hover',
	);

	// A primitive, so remapping the nodes (selection flags) fetches nothing again.
	const id = $derived(node.id);
	const settings = $derived(postItsApi.get_settings(id));
	const output = $derived(results.get(id));

	let refreshing = $state(false);

	async function refresh() {
		refreshing = true;
		try {
			await results.refresh(id);
		} catch {
			// The content section shows the error.
		} finally {
			refreshing = false;
		}
	}

	function labelOf(key: string) {
		return parameters[node.type ?? '']?.find((p) => p.key === key)?.label ?? key.replace(/^\$/, '');
	}

	function failure(err: unknown) {
		return isHttpError(err) && err.status === 403 ? m['node_detail.forbidden']() : m['node_detail.error']();
	}
</script>

{#snippet record(entries: [string, unknown][], empty: string)}
	{#if entries.length === 0}
		<p class="opacity-70 sm:text-sm md:text-base">{empty}</p>
	{:else}
	<!-- Display a list of key-value pairs: medium text for desktop, small text for mobile -->
		<dl class="flex flex-col gap-1 sm:text-sm md:text-base">
			{#each entries as [name, value] (name)}
				<div class="wrap-break-word">
					<dt class="inline font-semibold">{capitalize(name)}:</dt>
					<dd class="inline">
						{#if isLink(value)}
							<!-- An external address the card returned, not an app route. -->
							<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
							<a href={value} target="_blank" rel="noopener noreferrer" class="underline">
								{#if isImageLink(value)}
									<img src={value} alt={name} class="max-h-40 max-w-full" />
								{:else}
									{value}
								{/if}
							</a>
						{:else}
							{display(value) || '—'}
						{/if}
					</dd>
				</div>
			{/each}
		</dl>
	{/if}
{/snippet}

<aside
	class="absolute inset-0 z-25 flex flex-col gap-4 overflow-hidden bg-tertiary text-black md:static md:h-auto md:max-h-full md:min-h-0 md:self-stretch md:w-80 md:shrink-0 md:rounded-l-2xl md:border-l md:border-tertiary-border"
>
	<header class="flex shrink-0 items-start gap-2 p-4 pb-0">
		<button
			type="button"
			class="cursor-pointer rounded-md p-2 hover:bg-tertiary-hover"
			aria-label={m['node_detail.back']()}
			title={m['node_detail.back']()}
			onclick={onback}
		>
			<Icon name="left-arrow" class="stroke-2 w-5 h-5" />
		</button>

		<div class="flex min-w-0 flex-1 flex-col pt-1">
			<h2 class="text-lg font-semibold wrap-break-word">
				{#if title?.text && title.vars}
					{#await output}
						{title.text}
					{:then data}
						{renderTitle(title.text, data)}
					{:catch}
						{title.text}
					{/await}
				{:else}
					{title?.text || kind}
				{/if}
			</h2>
			{#if title?.text}
				<span class="text-xs opacity-70">{kind}</span>
			{/if}
		</div>

		<button
			type="button"
			class="cursor-pointer rounded-md p-2 hover:bg-tertiary-hover disabled:cursor-wait disabled:opacity-60"
			aria-label={m['node_detail.refresh']()}
			title={m['node_detail.refresh']()}
			disabled={refreshing}
			onclick={refresh}
		>
			<Icon name="refresh" class={'stroke-2 w-5 h-5 ' + (refreshing ? 'animate-spin' : '')} />
		</button>
	</header>

	<!-- Only the details scroll: the header and the delete button stay put. -->
	<div class={cn("flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4", scrollClass)}>
		<section class="flex flex-col gap-2">
			<h3 class="font-semibold">{m['node_detail.configuration']()}</h3>
			{#await settings}
				<p class="text-sm opacity-70">{m.input_loading()}</p>
			{:then { params }}
				{@render record(
					Object.entries(params ?? {}).map(([key, value]) => [labelOf(key), value]),
					m['node_detail.no_configuration']()
				)}
			{:catch err}
				<p class="text-sm text-destructive">{failure(err)}</p>
			{/await}
		</section>

		<section class="flex flex-col gap-2">
			<h3 class="font-semibold">{m['node_detail.content']()}</h3>
			{#await output}
				<p class="text-sm opacity-70">{m.input_loading()}</p>
			{:then data}
				{@render record(Object.entries(data ?? {}), m['node_detail.no_content']())}
			{:catch err}
				<p class="text-sm text-destructive">{failure(err)}</p>
			{/await}
		</section>
	</div>

	{#if canEdit}
		<div class="flex shrink-0 items-center justify-center p-4 pt-0">
			<Button variant="destructive" class="shrink-0" onclick={() => ondelete(node)}>
				{m['node_detail.delete']()}
			</Button>
		</div>
	{/if}
</aside>
