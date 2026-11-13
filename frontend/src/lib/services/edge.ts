import * as api from '$modules/api.svelte';
import type { Strand, StrandStyle, UUID } from '$types/api';

export async function connect(boardId: UUID, source: UUID, target: UUID, style: StrandStyle = {}): Promise<Strand> {
	const response = await api.post(`/v1/boards/${boardId}/strands`, { source, target, ...style });
	return await response.json() as Strand;
}

// Changes only the fields sent; an empty value goes back to the default.
export async function update(boardId: UUID, id: UUID, patch: StrandStyle): Promise<Strand> {
	const response = await api.patch(`/v1/boards/${boardId}/strands/${id}`, patch);
	if (!response.ok) throw new Error(`strand update failed: ${response.status}`);
	return await response.json() as Strand;
}

export async function disconnect(boardId: UUID, id: UUID) {
	await api.del(`/v1/boards/${boardId}/strands/${id}`);
}
