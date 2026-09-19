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
	onChange: (data: Update) => void
): Promise<Connection> {
	const id = uuid();
	const peers = await socket(board, id, onChange);
	return new RTC(id, peers);
}
