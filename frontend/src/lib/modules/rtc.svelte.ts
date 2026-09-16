import { Peer, type DataConnection } from 'peerjs';

import { mouses, type ClientData } from '$stores/mouses.svelte';

import { SvelteMap } from 'svelte/reactivity';

export class RTC {
	private static readonly COLORS = [
		'#FF0000',
		'#F0F000',
		'#00FF00',
		'#00F0F0',
		'#0000FF',
		'#F000F0'
	];

	private readonly connections = new SvelteMap<string, DataConnection>();
	private readonly peer: Peer;

	constructor(id: string, peers: string[]) {
		this.peer = new Peer(id);

		const color = RTC.COLORS[peers.length % RTC.COLORS.length];

		this.peer.on('connection', this.setPeerConnection.bind(this));
		this.peer.on('error', console.error);

		this.peer.on('open', () =>
			peers.forEach((p) => {
				const conn = this.peer.connect(p, {
					reliable: true,
					metadata: {
						username: 'Messi',
						picture: 'TBD',
						color
					} satisfies ClientData
				});

				this.setPeerConnection(conn);
			})
		);
	}

	update(data: [number, number]) {
		this.connections
			.values()
			.filter((c) => c.open)
			.forEach((c) => c.send(data));
	}

	close() {
		this.peer.destroy();
		this.connections.clear();
		mouses.clear();
	}

	private setPeerConnection(conn: DataConnection) {
		const id = conn.peer;

		conn.on('open', () => {
			this.connections.set(id, conn);
			mouses.add(id, conn.metadata);

			conn.on('data', (data) => {
				if (Array.isArray(data) && RTC.isNumber(data[0]) && RTC.isNumber(data[1])) {
					const pos = { x: data[0], y: data[1] };
					mouses.update(id, pos);
				}
			});
		});

		conn.on('close', () => {
			this.connections.delete(id);
			mouses.remove(id);

			console.log('Lost', id);
		});

		conn.on('error', console.error);
	}

	private static isNumber(n: unknown): n is number {
		return Number.isFinite(n);
	}
}
