import * as api from '$modules/api.svelte';
import type { Locale } from '$lib/paraglide/runtime';
import type { Profile, UserSummary, UUID } from '$types/api';

// Your own id answers the full profile; anyone else's only the summary.
export async function get(id: UUID) {
	const response = await api.get(`/v1/users/${id}`);
	return (await response.json()) as Profile | UserSummary;
}

// Only the fields sent change.
export async function update(id: UUID, changes: { name?: string; locale?: Locale }) {
	const response = await api.patch(`/v1/users/${id}`, changes);
	if (!response.ok) throw new Error(`profile update failed: ${response.status}`);
	return (await response.json()) as Profile;
}
