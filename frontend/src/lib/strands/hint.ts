// The hint for the first strand. Drawing one (dragging from a card's
// edge onto another) is the gesture people did not find; the hint shows on a
// board someone may edit, with two cards or more and no strand yet, until
// they draw one or dismiss it, once per browser and user.

type Storage = Pick<globalThis.Storage, 'getItem' | 'setItem'>;

export const strandHintKey = (userId: string) => `tiu.hint.strand:${userId}`;

export function showStrandHint(board: {
	canEdit: boolean;
	cards: number;
	strands: number;
	seen: boolean;
}): boolean {
	return board.canEdit && !board.seen && board.cards >= 2 && board.strands === 0;
}

function storage(): Storage | undefined {
	try {
		return globalThis.localStorage;
	} catch {
		return undefined;
	}
}

/** Whether the user already drew a strand or dismissed the hint here. */
export function strandHintSeen(userId: string | null | undefined, store = storage()): boolean {
	if (!userId) return true;
	try {
		return store?.getItem(strandHintKey(userId)) === '1';
	} catch {
		// Storage refused (a private window): showing it again is harmless.
		return false;
	}
}

export function markStrandHintSeen(userId: string | null | undefined, store = storage()) {
	if (!userId) return;
	try {
		store?.setItem(strandHintKey(userId), '1');
	} catch {
		// Nothing to keep it in: it shows again on the next visit.
	}
}
