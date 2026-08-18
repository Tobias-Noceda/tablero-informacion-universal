import * as api from '$modules/api.svelte';

export const connect = async (boardId: string, source: string, target: string) => {
    await api.post(`/v1/boards/${boardId}/connect`, { source, target });
}