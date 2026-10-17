import { writable } from 'svelte/store';
import type { Board } from '$types/api';
import * as boardService from '$services/board';
import { session } from '$modules/session.svelte';

const boards = writable<Board[]>([]);
// Bumped when the user changes, so a list requested for the previous one is dropped.
let generation = 0;

export async function refreshBoards() {
	const requested = generation;
	const list = await boardService.get_all();
	if (requested === generation) boards.set(list);
}

export function resetBoards() {
	generation++;
	boards.set([]);
}

session.onUserChange(resetBoards);

export const boardList = { subscribe: boards.subscribe };
