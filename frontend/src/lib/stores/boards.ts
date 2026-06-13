import { writable } from 'svelte/store';
import type { Board } from '$types/api';
import * as boardService from '$services/board';

const boards = writable<Board[]>([]);

export async function refreshBoards() {
	const list = await boardService.get_all();
	boards.set(list);
}

export const boardList = { subscribe: boards.subscribe };
