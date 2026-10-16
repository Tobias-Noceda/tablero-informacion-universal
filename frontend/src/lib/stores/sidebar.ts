import { writable } from "svelte/store";
import { session } from "$modules/session.svelte";

const sidebarOpen = writable<'cards' | 'credentials' | null>(null);
const managingSecrets = writable(false);
const sharing = writable(false);

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

session.onUserChange(() => {
    sidebarOpen.set(null);
    managingSecrets.set(false);
    sharing.set(false);
});
