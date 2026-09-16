import type { Board } from '$types/api';

import { io } from 'socket.io-client';

export type Update = { board: Board; ts: Date };

export async function socket(board: string, peer: string, update: (data: Update) => void) {
	const socket = io(
		import.meta.env.VITE_API_URL || window?.location.href || 'http://localhost:62113',
		{
			path: '/ws',
			transports: ['websocket'],
			query: {
				peer,
				board
			}
		}
	);

	const peers = await new Promise<string[]>((resolve, reject) => {
		socket.on('connect_error', reject);
		socket.once('peers', resolve);
	});

	socket.on('update', update);

	return peers;
}
