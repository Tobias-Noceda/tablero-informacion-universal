import { uuid } from '$lib/utils';
import type { Profile } from '$types/api';

import { RTC, type NodeMoves } from './rtc.svelte';
import { socket, type Update } from './sockets.svelte';

export type { Update } from './sockets.svelte';
export type { NodeMoves } from './rtc.svelte';

export interface Connection {
	update(update: [number, number]): void;
	moveNodes(moves: NodeMoves): void;
	close(): void;
}

export async function connect(
	board: string,
	user: Pick<Profile, 'id' | 'name' | 'picture'>,
	onChange: (data: Update) => void,
	onNodeMoves: (moves: NodeMoves) => void
): Promise<Connection> {
	const id = uuid();
	const live = await socket(board, id, onChange);
	const rtc = new RTC(id, live.clients, { username: user.name, picture: user.picture ?? '' }, onNodeMoves);
	return {
		update: (position) => rtc.update(position),
		moveNodes: (moves) => rtc.moveNodes(moves),
		close() {
			live.close();
			rtc.close();
		}
	};
}
