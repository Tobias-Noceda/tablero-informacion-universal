import { get, writable } from "svelte/store";
import { session } from "$modules/session.svelte";
import type { Draft } from "$lib/well-knowns/draft";

const sidebarOpen = writable<'cards' | 'credentials' | null>(null);
const managingSecrets = writable(false);
const sharing = writable(false);
const configuringBoard = writable(false);
const managingCardTypes = writable(false);

export const setSidebar = (type: 'cards' | 'credentials' | null) => {
    sidebarOpen.set(type);
}

export const getSidebar = { subscribe: sidebarOpen.subscribe };

export const setManagingSecrets = (value: boolean) => {
    managingSecrets.set(value);
}

export const isManagingSecrets = { subscribe: managingSecrets.subscribe };

export const setSharing = (value: boolean) => {
    sharing.set(value);
}

export const isSharing = { subscribe: sharing.subscribe };

export const setConfiguringBoard = (value: boolean) => {
    configuringBoard.set(value);
}

export const isConfiguringBoard = { subscribe: configuringBoard.subscribe };

export const setManagingCardTypes = (value: boolean) => {
    managingCardTypes.set(value);
}

export const isManagingCardTypes = { subscribe: managingCardTypes.subscribe };

// A new card type to start from (a one-off card saved as a type), taken
// once by the card types panel when it opens.
const cardTypeSeed = writable<Draft | null>(null);

export const startCardTypeFrom = (draft: Draft) => {
    cardTypeSeed.set(draft);
    managingCardTypes.set(true);
}

export const takeCardTypeSeed = (): Draft | null => {
    const seed = get(cardTypeSeed);
    cardTypeSeed.set(null);
    return seed;
}

session.onUserChange(() => {
    sidebarOpen.set(null);
    managingSecrets.set(false);
    sharing.set(false);
    configuringBoard.set(false);
    managingCardTypes.set(false);
    cardTypeSeed.set(null);
});
