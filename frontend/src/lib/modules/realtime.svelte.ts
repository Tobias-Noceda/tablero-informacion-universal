import type { Board } from '$types/api';

import { io } from 'socket.io-client';
import { type DataConnection, Peer } from 'peerjs';

import { mouses, type ClientData } from '$stores/mouses.svelte';

import { SvelteMap } from 'svelte/reactivity';

export type Update = { board: Board; ts: Date };

const colors = ['#FF0000', '#F0F000', '#00FF00', '#00F0F0', '#0000FF', '#F000F0'];

export async function connect(board: string, update: (data: Update) => void) {
	const id = crypto.randomUUID();

	const peers = await socket(board, id, update);
	const peer = rtc(id, peers);

	return () => {
		peer.destroy();
		connections.clear();
		mouses.clear();
	};
}

const connections = new SvelteMap<string, DataConnection>();

export function update(data: [number, number]) {
	connections
		.values()
		.filter((c) => c.open)
		.forEach((c) => c.send(data));
}

async function socket(board: string, peer: string, update: (data: Update) => void) {
	const socket = io('http://localhost:3000', {
		path: '/ws',
		transports: ['websocket'],
		query: {
			peer,
			board
		}
	});

	const peers = await new Promise<string[]>((resolve, reject) => {
		socket.on('connect_error', reject);
		socket.once('peers', resolve);
	});

	socket.on('update', update);

	return peers;
}

function rtc(id: string, peers: string[]) {
	const peer = new Peer(id);
	const color = colors[peers.length % colors.length];

	peer.on('connection', setPeerConnection);
	peer.on('error', console.error);

	peer.on('open', () =>
		peers.forEach((p) => {
			const conn = peer.connect(p, {
				reliable: true,
				metadata: {
					username: 'Messi',
					picture: 'TBD',
					color
				} satisfies ClientData
			});

			setPeerConnection(conn);
		})
	);

	return peer;
}

function setPeerConnection(conn: DataConnection) {
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

function isNumber(n: unknown): n is number {
	return Number.isFinite(n);
}
