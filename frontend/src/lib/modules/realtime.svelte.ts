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
	const live = await socket(board, id, onChange);
	const rtc = new RTC(id, live.clients, { username: user.name, picture: user.picture ?? '' });
	return {
		update: (position) => rtc.update(position),
		close() {
			live.close();
			rtc.close();
		}
	};
}
