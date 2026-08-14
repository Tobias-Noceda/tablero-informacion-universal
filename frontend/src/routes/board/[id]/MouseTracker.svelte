<script lang="ts">
	import { type Snippet } from 'svelte';

	import { type DataConnection, Peer } from 'peerjs';

	import { mouses, type ClientData } from '$stores/mouses.svelte';

	import { SvelteMap } from 'svelte/reactivity';
	import { page } from '$app/state';

	let { children }: { children: Snippet } = $props();

	const id = crypto.randomUUID();

	const connections = new SvelteMap<string, DataConnection>();

	let frame: number | null = null;

	function isNumber(n: unknown): n is number {
		return Number.isFinite(n);
	}

	function setConnection(conn: DataConnection) {
		const id = conn.peer;

		conn.on('open', () => {
			connections.set(id, conn);
			mouses.add(id, conn.metadata);

			conn.on('data', (data) => {
				if (Array.isArray(data) && isNumber(data[0]) && isNumber(data[1])) {
					const pos = { x: data[0], y: data[1] };
					mouses.update(id, pos);
				}
			});
		});

		conn.on('close', () => {
			connections.delete(id);
			mouses.remove(id);

			console.log('Lost', id);
		});

		conn.on('error', console.error);
	}

	$effect(() => {
		const peer = new Peer(id);

		peer.on('connection', setConnection);

		peer.on('open', (id) => {
			console.log('My peer ID is', id);

			// TBD: Read the ids from a DB (MemCache maybe?)
			page.url.searchParams.getAll('peer').forEach((p) => {
				const conn = peer.connect(p, {
					reliable: true,
					metadata: {
						username: 'Messi',
						picture: 'TBD',
						color: '#FF0000'
					} satisfies ClientData
				});

				setConnection(conn);
			});
		});

		return () => {
			if (frame) cancelAnimationFrame(frame);

			connections
				.values()
				.filter((c) => c.open)
				.forEach((c) => c.close());

			peer.destroy();

			connections.clear();
			mouses.clear();
		};
	});

	function move(e: MouseEvent) {
		if (frame !== null) {
			return;
		}

		frame = requestAnimationFrame(() => {
			frame = null;

			const { x, y } = mouses.convert({ x: e.clientX, y: e.clientY });
			const data = [x, y];

			connections
				.values()
				.filter((c) => c.open)
				.forEach((c) => c.send(data));
		});
	}

	let width = $state(0);
	let height = $state(0);

	function visible(x: number, y: number) {
		const margin = 10;
		return -margin < x && x < width - margin && -margin < y && y < height - margin;
	}
</script>

<svelte:window bind:innerWidth={width} bind:innerHeight={height} />

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div onmousemove={move} class="contents">
	{@render children()}
</div>

{#each mouses.data() as mouse (mouse.username)}
	{#if mouse.position && visible(mouse.position.x, mouse.position.y)}
		<div
			class="size-2 absolute"
			style:background-color={mouse.color}
			style:top="{mouse.position.y}px"
			style:left="{mouse.position.x}px"
		></div>
	{/if}
{/each}
