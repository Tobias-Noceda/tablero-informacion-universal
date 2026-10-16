import * as api from '$modules/api.svelte';
import type { Profile, UserSummary, UUID } from '$types/api';

// Your own id answers the full profile; anyone else's only the summary.
export async function get(id: UUID) {
	const response = await api.get(`/v1/users/${id}`);
	return (await response.json()) as Profile | UserSummary;
}

export async function rename(id: UUID, name: string) {
	const response = await api.patch(`/v1/users/${id}`, { name });
	if (!response.ok) throw new Error(`rename failed: ${response.status}`);
	return (await response.json()) as Profile;
}
