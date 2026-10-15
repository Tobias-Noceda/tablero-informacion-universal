import * as orgsApi from '$services/orgs';
import type { Board, UUID, Org } from '$types/api';

// The switcher's value for boards under no organization.
export const PERSONAL = 'personal';

const STORAGE_KEY = 'tiu.org';

function remembered(): UUID | undefined {
	try {
		return localStorage.getItem(STORAGE_KEY) || undefined;
	} catch {
		return undefined;
	}
}

// The organizations the user is in. Which one is being looked at comes from
// the URL (/organizations/[orgId]/...), "/" and /boards/[id] being Personal.
class Organizations {
	list = $state<Org[]>([]);
	#loading: Promise<void> | undefined;
	#last = $state<UUID | undefined>(remembered());

	// The organization of the last landing or board opened, for pages that are
	// about none (profile, organization management). Undefined for Personal,
	// or once the user is no longer in it. Kept per browser in localStorage.
	get last() {
		return this.byId(this.#last)?.id;
	}

	remember(orgId: UUID | undefined) {
		this.#last = orgId;
		try {
			if (orgId) localStorage.setItem(STORAGE_KEY, orgId);
			else localStorage.removeItem(STORAGE_KEY);
		} catch {
			// Storage may be unavailable; the choice then lasts until a reload.
		}
	}

	// Loads the list once; route loads await it before deciding a 404.
	ensure() {
		this.#loading ??= orgsApi.list().then(
			(list) => {
				this.list = list;
			},
			(err) => {
				this.#loading = undefined;
				throw err;
			}
		);
		return this.#loading;
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
