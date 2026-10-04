import { uuid } from '$lib/utils';
import type { Profile } from '$types/api';

import { RTC } from './rtc.svelte';
import { socket, type Update } from './sockets.svelte';

export type { Update } from './sockets.svelte';

export interface Connection {
	update(update: [number, number]): void;
	close(): void;
}

export async function connect(
	board: string,
	user: Pick<Profile, 'id' | 'name' | 'picture'>,
	onChange: (data: Update) => void
): Promise<Connection> {
	const id = uuid();
	const clients = await socket(board, user.id, id, onChange);
	return new RTC(id, clients, { username: user.name, picture: user.picture ?? '' });
}
