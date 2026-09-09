import type { Board } from '$types/api';

import { io } from 'socket.io-client';

export type Update = { board: Board; ts: Date };

export async function connect(board: string, peer: string, update: (data: Update) => void) {
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
