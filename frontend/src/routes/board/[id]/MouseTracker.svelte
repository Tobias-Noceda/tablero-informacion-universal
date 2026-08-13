<script lang="ts">
	import { type Snippet } from 'svelte';

	import { type DataConnection, Peer } from 'peerjs';

	import { mouses } from '$stores/mouses.svelte';

	import { SvelteMap } from 'svelte/reactivity';
	import { page } from '$app/state';

	let { children }: { children: Snippet } = $props();

	const id = crypto.randomUUID();

	const connections = new SvelteMap<string, DataConnection>();

	let frame: number | null = null;

	function isNumber(n: unknown): n is number {
		return Number.isSafeInteger(n);
	}

	function setConnection(conn: DataConnection) {
		const id = conn.connectionId;

		conn.on('open', () => {
			connections.set(id, conn);
			mouses.add(id, {
				username: conn.label,
				picture: 'TBD'
			});

			conn.on('data', (data) => {
				console.log('Received', data);

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
					label: 'username_' + id,
					reliable: true
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

	// eslint-disable-next-line svelte/no-inspect
	$inspect(mouses).with(console.trace);
</script>

<!-- svelte-ignore a11y_no_static_element_interactions -->
<div onmousemove={move}>
	{@render children()}
</div>

{#each mouses.data() as mouse (mouse.username)}
	{#if mouse.position}
		<p>{mouse.username}: {mouse.position.x} {mouse.position.y}</p>
		<div
			class="size-2 absolute bg-red-700"
			style:top="{mouse.position.y}px"
			style:right="{mouse.position.x}px"
		></div>
	{:else}
		<p>{mouse.username}: Loading...</p>
	{/if}
{/each}
