import * as api from '$modules/api.svelte';
import type { Strand, UUID } from '$types/api';

export async function connect(boardId: UUID, source: UUID, target: UUID): Promise<Strand> {
	const response = await api.post(`/v1/boards/${boardId}/strands`, { source, target });
	return await response.json() as Strand;
}

export async function disconnect(boardId: UUID, id: UUID) {
	await api.del(`/v1/boards/${boardId}/strands/${id}`);
}
