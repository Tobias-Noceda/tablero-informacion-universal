import { writable } from "svelte/store";
import { session } from "$modules/session.svelte";

const creatingBoard = writable(false);

export const setCreatingBoard = (value: boolean) => {
    creatingBoard.set(value);
}

export const isCreatingBoard = { subscribe: creatingBoard.subscribe };

session.onUserChange(() => creatingBoard.set(false));
