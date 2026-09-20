import { uuid } from '$lib/utils';

import { RTC } from './rtc.svelte';
import { socket, type Update } from './sockets.svelte';

export type { Update } from './sockets.svelte';

export interface Connection {
	update(update: [number, number]): void;
	close(): void;
}

export async function connect(
	board: string,
	user: string,
	onChange: (data: Update) => void
): Promise<Connection> {
	const id = uuid();
	const clients = await socket(board, user, id, onChange);
	return new RTC(id, clients);
}
