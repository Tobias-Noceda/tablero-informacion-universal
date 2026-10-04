import type { Board } from '$types/api';

import { io, type Socket } from 'socket.io-client';

import { session } from './session.svelte';

export type Update = { board: Board; ts: Date };
// Who else is on the board: peer id → user id.
export type Clients = Record<string, string>;

export type BoardSocket = {
	clients: Clients;
	close(): void;
};

// A token valid for longer than this was already renewed elsewhere.
const FRESH_FOR_MS = 90_000;

async function freshToken() {
	if (session.expiresAt - Date.now() < FRESH_FOR_MS && !(await session.refresh())) return null;
	return session.accessToken;
}

// A refused handshake is retried once with a renewed session, then given up:
// the board keeps working without live updates.
export async function socket(
	board: string,
	peer: string,
	update: (data: Update) => void
): Promise<BoardSocket> {
	const socket: Socket = io(
		import.meta.env.VITE_REALTIME_URL ||
			import.meta.env.VITE_API_URL ||
			window?.location.origin ||
			'http://localhost:3000',
		{
			path: '/ws',
			transports: ['websocket'],
			query: { board, peer },
			auth: (send) => send({ token: session.accessToken ?? '' })
		}
	);

	let retried = false;
	let giveUp = (err: Error) => console.warn('Live updates stopped:', err.message);

	socket.on('connect', () => (retried = false));

	socket.on('connect_error', async (err) => {
		// Network failures are retried by socket.io; refusals are not.
		if (socket.active) return;
		if (err.message === 'unauthorized' && !retried) {
			retried = true;
			if (await session.refresh()) return void socket.connect();
		}
		giveUp(err);
	});

	socket.on('disconnect', (reason) => {
		if (reason === 'io server disconnect') socket.connect();
	});

	socket.on('token_expiring', async () => {
		const token = await freshToken();
		if (token) socket.emit('auth', { token });
	});

	socket.on('update', update);

	const clients = await new Promise<Clients>((resolve, reject) => {
		giveUp = reject;
		socket.once('peers', resolve);
	})
		.catch((err) => {
			socket.close();
			throw err;
		})
		.finally(() => {
			giveUp = (err) => console.warn('Live updates stopped:', err.message);
		});

	return { clients, close: () => socket.close() };
}
