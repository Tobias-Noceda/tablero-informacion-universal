import type { SecretMeta, UUID } from '$types/api';

// Where a usable secret comes from, as the picker groups it. "shared" is
// anything the user reaches only through a grant.
export type Origin = 'board' | 'mine' | 'profile' | 'group' | 'shared';

const ORDER: Origin[] = ['board', 'mine', 'profile', 'group', 'shared'];

export function originOf(secret: SecretMeta, board: UUID, me: string): Origin {
	const { kind, owner } = secret.scope;

	if (kind === 'board' && owner === board) return 'board';
	if (kind === 'member' && owner === `${board}:${me}`) return 'mine';
	if (kind === 'user' && owner === me) return 'profile';
	if (kind === 'group') return 'group';
	return 'shared';
}

export function groupByOrigin(usable: SecretMeta[], board: UUID, me: string) {
	const buckets = new Map<Origin, SecretMeta[]>();
	for (const secret of usable) {
		const origin = originOf(secret, board, me);
		buckets.set(origin, [...(buckets.get(origin) ?? []), secret]);
	}

	return ORDER.filter((origin) => buckets.has(origin)).map((origin) => ({
		origin,
		secrets: buckets.get(origin)!
	}));
}
