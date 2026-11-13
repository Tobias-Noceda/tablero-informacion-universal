<script lang="ts">
	import { isHttpError } from '@sveltejs/kit';
	import type { Node } from '@xyflow/svelte';

	import Button from '$components/Button/Button.svelte';
	import Icon from '$components/Icon/Icon.svelte';
	import { wellKnowns } from '$stores/well-knowns.svelte';
	import { m } from '$lib/paraglide/messages';
	import { display, renderTitle } from '$lib/title';
	import * as postItsApi from '$services/post-it';
	import { results } from '$stores/postit-results.svelte';
	import { cn } from '$lib/utils';
	import {
		capitalize,
		githubRepoPath,
		githubRepoUrl,
		isImageLink,
		isLink
	} from '$lib/helpers/content-rendering';
	import { getLocale } from '$lib/paraglide/runtime';
	import Tooltip from '$components/Tooltip/Tooltip.svelte';
	import Switch from '$components/Switch/Switch.svelte';
	import { renderMarkdown } from '$lib/helpers/markdown';
	import type { PostIt, PostItSettings, SecretRef } from '$types/api';
	import { ONE_OFF } from '$lib/well-knowns/one-off';

	export type PostItData = PostIt & {
		title?: { text: string; vars: boolean };
		paramValues: Record<string, string>;
		bindings: Record<string, SecretRef>;
		/** What the card is stored with; a one-off card's form needs it all. */
		settings?: PostItSettings;
	};

	interface NodeDetailProps {
		node: Node;
		canEdit: boolean;
		canDelete: boolean;
		settingsVersion: number;
		onback: () => void;
		onedit: (node: PostItData) => void;
		ondelete: (node: Node) => void;
	}

	let { node, canEdit, canDelete, settingsVersion, onback, onedit, ondelete }: NodeDetailProps = $props();

	const title = $derived(node.data?.title as { text: string; vars: boolean } | undefined);

	// What kind of card it is, as the cards sidebar names it.
	const kind = $derived(wellKnowns.name(node.type));
	// A card whose well-known was deleted keeps running, but there is no form
	// left to edit it with.
	// A one-off card carries its own request, so it is always editable.
	const isOneOff = $derived(node.type === ONE_OFF);
	const knownType = $derived(isOneOff || !!wellKnowns.byKey(node.type));

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
	// Kept up to date while open, also for a card whose title needs no data.
	$effect(() => results.watch(id));

	// A note shows its raw text until asked to render it; back to raw on
	// every card opened (the panel stays mounted while switching cards).
	let markdown = $derived.by(() => {
		void id;
		return false;
	});

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

	function parameterLabelOf(key: string) {
		return (
			wellKnowns.parameters(node.type).find((p) => p.key === key)?.[`label_${getLocale()}`] ??
			key.replace(/^\$/, '')
		);
	}

	// A configuration row: label, value and, for some, the page it links to.
	// The stored value is left as it is; only what the panel shows changes.
	function configurationEntry(key: string, value: unknown): [string, unknown, string?] {
		const label = parameterLabelOf(key);
		if (node.type === 'github_repo' && key === '$query') {
			const href = githubRepoUrl(value);
			if (href) return [label, githubRepoPath(value), href];
		}
		return [label, value];
	}

	function contentLabelOf(key: string) {
		return (
			wellKnowns.outputs(node.type).find((c) => c.key === key)?.[`label_${getLocale()}`] ?? key
		);
	}

	function failure(err: unknown) {
		return isHttpError(err) && err.status === 403
			? m['node_detail.forbidden']()
			: m['node_detail.error']();
	}
</script>

{#snippet record(entries: [string, unknown, string?][], empty: string)}
	{#if entries.length === 0}
		<p class="opacity-70 sm:text-sm md:text-[15px]">{empty}</p>
	{:else}
		<!-- Display a list of key-value pairs: medium text for desktop, small text for mobile -->
		<dl class="flex flex-col gap-1 sm:text-sm md:text-[15px]">
			{#each entries as [name, value, href] (name)}
				<div class="wrap-break-word">
					<dt class="inline font-semibold">{capitalize(name)}:</dt>
					<dd class="inline">
						{#if href}
							<Tooltip content={href} placement="bottom" tooltipClass="max-w-60! break-all">
								<!-- An external page the value names, not an app route. -->
								<!-- eslint-disable-next-line svelte/no-navigation-without-resolve -->
								<a {href} target="_blank" rel="noopener noreferrer" class="underline">
									{display(value)}
								</a>
							</Tooltip>
						{:else if isLink(value)}
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

<!-- A one-off card's own request. A credential shows as its name, never its value. -->
{#snippet oneOffConfiguration(s: PostItSettings)}
	<div class="flex flex-col gap-3">
		{@render record(
			[
				[m['one_off.url'](), s.resource ?? ''],
				[m['one_off.response'](), (s.response ?? '').toUpperCase()],
				[m['one_off.rate'](), m['one_off.seconds']({ value: String(s.rate) })]
			],
			m['node_detail.no_configuration']()
		)}
		<!-- {#each [[m['postit_templates.queries'](), s.request?.queries], [m['postit_templates.headers'](), s.request?.headers]] as const as [label, values] (label)}
			{#if values && Object.keys(values).length > 0}
				<div class="flex flex-col gap-1">
					<h4 class="font-semibold text-sm">{label}</h4>
					{@render record(
						Object.entries(values).map(([key, value]) => [key, shown(value)]),
						''
					)}
				</div>
			{/if}
		{/each}
		<div class="flex flex-col gap-1">
			<h4 class="font-semibold text-sm">{m['one_off.outputs']()}</h4>
			{@render record(Object.entries(s.query ?? {}), m['node_detail.no_configuration']())}
		</div> -->
	</div>
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
			</h2>
		</Tooltip>
	{:else}
		<h2 class="text-lg font-semibold whitespace-wrap line-clamp-1">{finalTitle}</h2>
	{/if}
{/snippet}

<aside
	class="absolute inset-0 z-25 flex flex-col gap-4 overflow-hidden bg-detail text-detail-text md:static md:h-auto md:max-h-full md:min-h-0 md:self-stretch md:w-100 md:shrink-0 md:rounded-l-2xl md:border-l md:border-detail-border"
>
	<header class="flex shrink-0 items-start gap-2 p-4 pb-0">
		<Tooltip
			content={m['node_detail.back']()}
			placement="bottom"
			tooltipClass="max-w-60! -mt-1 -ml-9"
		>
			<button
				type="button"
				class="cursor-pointer rounded-md p-2 hover:bg-detail-hover"
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
			{#if canEdit && !knownType}
				<span class="text-xs opacity-70">{m['node_detail.unavailable_type']()}</span>
			{/if}
		</div>

		{#if canEdit && knownType}
			<Tooltip content={m['node_detail.edit']()} placement="bottom" tooltipClass="max-w-60! -mt-1">
				<button
					type="button"
					class="cursor-pointer rounded-md p-2 hover:bg-detail-hover disabled:cursor-wait disabled:opacity-60"
					aria-label={m['node_detail.edit']()}
					onclick={async () => {
						const s = await settings;
						onedit({
							...node,
							title,
							paramValues: s.params ?? {},
							bindings: s.bindings ?? {},
							settings: s
						});
					}}
				>
					<Icon name="edit" class="stroke-2 w-5 h-5" />
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
				class="cursor-pointer rounded-md p-2 hover:bg-detail-hover disabled:cursor-wait disabled:opacity-60"
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
		{#if node.type !== 'static_card'}
			<section class="flex flex-col gap-2">
				<h3 class="font-semibold text-xl">{m['node_detail.configuration']()}</h3>
				<div class="text-xs">
					{#await settings}
						<p class="text-sm opacity-70">{m.input_loading()}</p>
					{:then s}
						{#if isOneOff}
							{@render oneOffConfiguration(s)}
						{:else}
							{@render record(
								Object.entries(s.params ?? {}).map(([key, value]) => configurationEntry(key, value)),
								m['node_detail.no_configuration']()
							)}
						{/if}
					{:catch err}
						<p class="text-sm text-destructive">{failure(err)}</p>
					{/await}
				</div>
			</section>
		{/if}

		<section class="flex flex-col gap-2">
			<div class="flex items-center justify-between gap-2">
				<h3 class="font-semibold text-xl">{m['node_detail.content']()}</h3>
				{#if node.type === 'static_card'}
					<Switch
						label={m['node_detail.render_markdown']()}
						checked={markdown}
						onchange={(on) => (markdown = on)}
					/>
				{/if}
			</div>
			{#await output}
				<p class="text-sm opacity-70">{m.input_loading()}</p>
			{:then data}
				{#if node.type === 'static_card'}
					{#if markdown}
						<div
							class="prose prose-sm max-w-none wrap-break-word text-inherit [--tw-prose-body:currentColor] [--tw-prose-bold:currentColor] [--tw-prose-bullets:currentColor] [--tw-prose-code:currentColor] [--tw-prose-counters:currentColor] [--tw-prose-headings:currentColor] [--tw-prose-links:currentColor] [--tw-prose-quotes:currentColor]"
						>
							<!-- Sanitized by renderMarkdown (DOMPurify): notes come from other users. -->
							<!-- eslint-disable-next-line svelte/no-at-html-tags -->
							{@html renderMarkdown(data?.text)}
						</div>
					{:else}
						<p class="whitespace-pre-wrap wrap-break-word sm:text-sm md:text-[15px]">{data?.text}</p>
					{/if}
				{:else}
					{@render record(
						Object.entries(data ?? {}).map(([key, value]) => [contentLabelOf(key), value]),
						m['node_detail.no_content']()
					)}
				{/if}
			{:catch err}
				<p class="text-sm text-destructive">{failure(err)}</p>
			{/await}
		</section>
	</div>

	{#if canDelete}
		<div class="flex shrink-0 items-center justify-center p-4 pt-0">
			<Button variant="destructive" class="shrink-0" onclick={() => ondelete(node)}>
				{m['node_detail.delete']()}
			</Button>
		</div>
	{/if}
</aside>
