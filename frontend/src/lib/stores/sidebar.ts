import { writable } from "svelte/store";

const sidebarOpen = writable<'cards' | 'credentials' | null>(null);
const managingSecrets = writable(false);

export const setSidebar = (type: 'cards' | 'credentials' | null) => {
    sidebarOpen.set(type);
}

export const getSidebar = { subscribe: sidebarOpen.subscribe };

export const setManagingSecrets = (value: boolean) => {
    managingSecrets.set(value);
}

export const isManagingSecrets = { subscribe: managingSecrets.subscribe };