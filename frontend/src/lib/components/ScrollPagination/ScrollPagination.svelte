<script lang="ts" generics="T">
	import type { Paginated } from '$lib/types/api';
	import { onMount, type Snippet } from 'svelte';

	import { afterNavigate, beforeNavigate } from '$app/navigation';
	import { preserve, restore } from '$modules/statefull.svelte';

	interface Props {
		initialItems: Paginated<T>;
		/**
		 * The function to fetch the next page.
		 */
		nextFetchFunction: (nextUrl: string) => Promise<Paginated<T>>;
		/**
		 * Stable key used for the list rendering.
		 */
		key?: (item: T, index: number) => string | number;
		/**
		 * The component to render per element.
		 */
		children: Snippet<[T, number]>;
		/**
		 * The loading component to render while fetching the next page.
		 */
		loading: Snippet;
		/**
		 * The error component to render if an error occurs.
		 */
		error?: Snippet;
	}

	let {
		initialItems,
		nextFetchFunction,
		key = (_, index) => index,
		children,
		loading,
		error,
	}: Props = $props();

	let additionalEntries = $state<T[]>([]);
	let entries = $derived([...initialItems.results, ...additionalEntries]);

	let load = $state(false);
	let done = $state(false);
	let erro = $state(false);

	let next: string | undefined = $derived(initialItems._links?.next);

	async function getPage() {
		if (!next) {
			done = true;
			return;
		}
		load = true;

		await nextFetchFunction(next)
			.then((response) => {
				additionalEntries.push(...response.results);
				entries = [...initialItems.results, ...additionalEntries];
				next = response._links?.next;
			})
			.catch((e) => {
				console.error('Error in fetch:', e);
				erro = true;
				throw e;
			})
			.finally(() => {
				load = false;

				if (!next) {
					observer.disconnect();
					done = true;
				}
			});
	}

	let sentinel: HTMLElement | null = $state(null);
	let observer: IntersectionObserver;
	onMount(() => {
		observer = new IntersectionObserver(
			(entries) => {
				if (entries[0].isIntersecting) getPage();
			},
			{
				root: null,
				rootMargin: '100px',
				threshold: 1.0
			}
		);

		observer.observe(sentinel!);
	});

	let scrollY: number = $state(0);

	type State = {
		next: string | undefined;
		load: boolean;
		done: boolean;
		entries: T[];
		additionalEntries: T[];
		scrollY: number;
	};

	beforeNavigate(({ from, to }) => {
		preserve(from, to, {
			next,
			load,
			done,
			entries,
			additionalEntries,
			scrollY
		} satisfies State);
	});

	afterNavigate(({ from, to }) => {
		const data = restore<State>(from, to);

		if (data) {
			next = data.next;
			load = data.load;
			done = data.done;
			entries = data.entries;
			additionalEntries = data.additionalEntries;

			setTimeout(() => (scrollY = data.scrollY), 500);
		}
	});
</script>

<svelte:window bind:scrollY />

{#each entries as e, i (key(e, i))}
	{@render children(e, i)}
{/each}

{#if !done}
	<div bind:this={sentinel} class="h-0.5" data-testid="done-test"></div>
{/if}

{#if load}
	{@render loading()}
{/if}

{#if erro && error}
	{@render error()}
{/if}