import { writable } from "svelte/store";

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
