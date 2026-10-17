import { session } from '$modules/session.svelte';
import * as orgsApi from '$services/orgs';
import type { Board, UUID, Org } from '$types/api';

// The switcher's value for boards under no organization.
export const PERSONAL = 'personal';

// One entry per user, so another account signing in on this browser does
// not start in the previous one's organization.
const storageKey = (userId: UUID) => `tiu.org:${userId}`;

function remembered(userId: UUID | null | undefined): UUID | undefined {
	if (!userId) return undefined;
	try {
		return localStorage.getItem(storageKey(userId)) || undefined;
	} catch {
		return undefined;
	}
}

// The organizations the user is in. Which one is being looked at comes from
// the URL (/organizations/[orgId]/...), "/" and /boards/[id] being Personal.
class Organizations {
	list = $state<Org[]>([]);
	#loading: Promise<void> | undefined;
	// Bumped on reset, so a list requested for the previous user is dropped.
	#generation = 0;
	#last = $state<UUID | undefined>(remembered(session.user?.id));

	// The organization of the last landing or board opened, for pages that are
	// about none (profile, organization management). Undefined for Personal,
	// or once the user is no longer in it. Kept per browser and user in localStorage.
	get last() {
		return this.byId(this.#last)?.id;
	}

	remember(orgId: UUID | undefined) {
		this.#last = orgId;
		const userId = session.user?.id;
		if (!userId) return;
		try {
			if (orgId) localStorage.setItem(storageKey(userId), orgId);
			else localStorage.removeItem(storageKey(userId));
		} catch {
			// Storage may be unavailable; the choice then lasts until a reload.
		}
	}

	// Loads the list once; route loads await it before deciding a 404.
	ensure() {
		const generation = this.#generation;
		this.#loading ??= orgsApi.list().then(
			(list) => {
				if (generation === this.#generation) this.list = list;
			},
			(err) => {
				if (generation === this.#generation) this.#loading = undefined;
				throw err;
			}
		);
		return this.#loading;
	}

	// Forgets everything about the previous user; the next ensure() loads anew.
	reset(userId: UUID | null) {
		this.#generation++;
		this.#loading = undefined;
		this.list = [];
		this.#last = remembered(userId);
	}

	async refresh() {
		this.#loading = undefined;
		await this.ensure();
	}

	byId(id: UUID | undefined) {
		return id ? this.list.find((org) => org.id === id) : undefined;
	}

	// The organization board is filed under, or undefined for Personal. A board
	// of an organization the user is not in (it was shared with them) is personal.
	orgOf(board: Board) {
		return board.org && this.byId(board.org) ? board.org : undefined;
	}

	// Whether board belongs under orgId (undefined for Personal).
	shows(board: Board, orgId?: UUID) {
		return this.orgOf(board) === orgId;
	}
}

export const orgs = new Organizations();
session.onUserChange((userId) => orgs.reset(userId));
