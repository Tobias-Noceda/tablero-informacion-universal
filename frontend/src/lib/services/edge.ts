import * as api from '$modules/api.svelte';
import type { UUID } from '$types/api';

export async function connect(id: UUID, source: UUID, target: UUID) {
	await api.post(`/v1/boards/${id}/strands`, { source, target });
}

// TODO: broken
export async function disconnect(id: UUID, source: UUID, target: UUID) {
	await api.del(`/v1/boards/${id}/strands`, { source, target });
}
