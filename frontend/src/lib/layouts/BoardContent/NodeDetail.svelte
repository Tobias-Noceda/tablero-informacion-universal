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
	import { getLocale } from '$lib/paraglide/runtime';
	import Tooltip from '$components/Tooltip/Tooltip.svelte';
	import type { PostIt, SecretRef } from '$types/api';

	export type PostItData = PostIt & {
		title?: { text: string; vars: boolean };
		paramValues: Record<string, string>;
		bindings: Record<string, SecretRef>;
	}

	interface NodeDetailProps {
		node: Node;
		canEdit: boolean;
		settingsVersion: number;
		onback: () => void;
		onedit: (node: PostItData) => void;
		ondelete: (node: Node) => void;
	}

	let {
		node,
		canEdit,
		settingsVersion,
		onback,
		onedit,
		ondelete
	}: NodeDetailProps = $props();

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
		'[&::-webkit-scrollbar-thumb]:hover:bg-main-hover'
	);

	// A primitive, so remapping the nodes (selection flags) fetches nothing again.
	const id = $derived(node.id);
	const settings = $derived.by(() => {
		void settingsVersion; // Refresh when the version changes.
		return postItsApi.get_settings(id);
	});
	const output = $derived(results.get(id));

	let refreshing = $state(false);
	const MAX_TITLE_LENGTH = 24;

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
		return (
			parameters[node.type ?? '']?.find((p) => p.key === key)?.[`label_${getLocale()}`] ??
			key.replace(/^\$/, '')
		);
	}

	function failure(err: unknown) {
		return isHttpError(err) && err.status === 403
			? m['node_detail.forbidden']()
			: m['node_detail.error']();
	}
</script>

{#snippet record(entries: [string, unknown][], empty: string)}
	{#if entries.length === 0}
		<p class="opacity-70 sm:text-sm md:text-[15px]">{empty}</p>
	{:else}
		<!-- Display a list of key-value pairs: medium text for desktop, small text for mobile -->
		<dl class="flex flex-col gap-1 sm:text-sm md:text-[15px]">
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

{#snippet titleHeading(finalTitle: string)}
	{#if finalTitle.length > MAX_TITLE_LENGTH}
		<Tooltip content={finalTitle} placement="bottom" tooltipClass="max-w-60!">
			<h2 class="text-lg font-semibold whitespace-wrap line-clamp-1" title={finalTitle}>
				{#if isLink(finalTitle)}
					<!-- An external address the card returned, not an app route. -->
					<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
					<a href={finalTitle} target="_blank" rel="noopener noreferrer" class="underline">
						{finalTitle}
					</a>
				{:else}
					{finalTitle}
				{/if}
				{finalTitle}
			</h2>
		</Tooltip>
	{:else}
		<h2 class="text-lg font-semibold whitespace-wrap line-clamp-1">{finalTitle}</h2>
	{/if}
{/snippet}

<aside
	class="absolute inset-0 z-25 flex flex-col gap-4 overflow-hidden bg-tertiary text-black md:static md:h-auto md:max-h-full md:min-h-0 md:self-stretch md:w-100 md:shrink-0 md:rounded-l-2xl md:border-l md:border-tertiary-border"
>
	<header class="flex shrink-0 items-start gap-2 p-4 pb-0">
		<Tooltip
			content={m['node_detail.back']()}
			placement="bottom"
			tooltipClass="max-w-60! -mt-1 -ml-9"
		>
			<button
				type="button"
				class="cursor-pointer rounded-md p-2 hover:bg-tertiary-hover"
				aria-label={m['node_detail.back']()}
				onclick={onback}
			>
				<Icon name="left-arrow" class="stroke-2 w-5 h-5" />
			</button>
		</Tooltip>

		<div class="flex min-w-0 flex-1 flex-col pt-1">
			{#if title?.text && title.vars}
				{#await output}
					{@render titleHeading(title.text)}
				{:then data}
					{@const finalTitle = renderTitle(title.text, data)}
					{@render titleHeading(finalTitle)}
				{:catch}
					{@render titleHeading(title.text)}
				{/await}
			{:else}
				{@render titleHeading(title?.text || kind)}
			{/if}
			{#if title?.text}
				<span class="text-xs opacity-70">{kind}</span>
			{/if}
		</div>

		{#if canEdit}
			<Tooltip
				content={m['node_detail.edit']()}
				placement="bottom"
				tooltipClass="max-w-60! -mt-1"
			>
				<button
					type="button"
					class="cursor-pointer rounded-md p-2 hover:bg-tertiary-hover disabled:cursor-wait disabled:opacity-60"
					aria-label={m['node_detail.edit']()}
					onclick={async () => {
						const s = await settings;
						onedit({
							...node,
							title,
							paramValues: s.params ?? {},
							bindings: s.bindings ?? {}
						})
					}}
				>
					<Icon name="edit" class='stroke-2 w-5 h-5' />
				</button>
			</Tooltip>
		{/if}

		<Tooltip
			content={m['node_detail.refresh']()}
			placement="bottom-left"
			tooltipClass="max-w-60! -mt-1 ml-11"
		>
			<button
				type="button"
				class="cursor-pointer rounded-md p-2 hover:bg-tertiary-hover disabled:cursor-wait disabled:opacity-60"
				aria-label={m['node_detail.refresh']()}
				disabled={refreshing}
				onclick={refresh}
			>
				<Icon name="refresh" class={'stroke-2 w-5 h-5 ' + (refreshing ? 'animate-spin' : '')} />
			</button>
		</Tooltip>
	</header>

	<!-- Only the details scroll: the header and the delete button stay put. -->
	<div class={cn('flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto px-4', scrollClass)}>
		<section class="flex flex-col gap-2">
			<h3 class="font-semibold text-xl">{m['node_detail.configuration']()}</h3>
			<div class="text-xs">
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
			</div>
		</section>

		<section class="flex flex-col gap-2">
			<h3 class="font-semibold text-xl">{m['node_detail.content']()}</h3>
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
