import type { Board } from '$types/api';

import { io } from 'socket.io-client';

export type Update = { board: Board; ts: Date };
export type Clients = Record<string, string>;

export async function socket(
	board: string,
	user: string,
	peer: string,
	update: (data: Update) => void
) {
	const socket = io(
		import.meta.env.VITE_REALTIME_URL ||
			import.meta.env.VITE_API_URL ||
			window?.location.origin ||
			'http://localhost:3000',
		{
			path: '/ws',
			transports: ['websocket'],
			query: {
				board,
				user,
				peer
			}
		}
	);

	const clients = await new Promise<Clients>((resolve, reject) => {
		socket.on('connect_error', reject);
		socket.once('peers', resolve);
	});

	socket.on('update', update);

	return clients;
}
