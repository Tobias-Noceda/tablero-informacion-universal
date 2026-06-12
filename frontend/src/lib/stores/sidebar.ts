import { writable } from "svelte/store";

const sidebarOpen = writable<boolean>(false);

export const toggleSidebar = () => {
    sidebarOpen.update(open => !open);
}

export const isSidebarOpen = { subscribe: sidebarOpen.subscribe };