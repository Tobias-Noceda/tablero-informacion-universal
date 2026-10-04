import * as orgsApi from '$services/orgs';
import type { Board, Org, UUID } from '$types/api';

const STORAGE_KEY = 'tiu.org';

export const PERSONAL = 'personal';
export type OrgSelection = UUID | typeof PERSONAL;

function remembered(): OrgSelection {
	try {
		return localStorage.getItem(STORAGE_KEY) || PERSONAL;
	} catch {
		return PERSONAL;
	}
}

// The organizations the user is in, and which one the sidebar and new boards
// are about. The choice is a per-browser convenience, kept in localStorage.
class Organizations {
	list = $state<Org[]>([]);
	selected = $state<OrgSelection>(remembered());

	// The selected organization, or undefined for "Personal".
	get current() {
		return this.list.find((org) => org.id === this.selected);
	}

	select(selection: OrgSelection) {
		this.selected = selection;
		try {
			localStorage.setItem(STORAGE_KEY, selection);
		} catch {
			// Storage may be unavailable; the choice then lasts until a reload.
		}
	}

	async refresh() {
		this.list = await orgsApi.list();
		if (this.selected !== PERSONAL && !this.current) this.select(PERSONAL);
	}

	// Whether board belongs under the current selection. A board of an
	// organization the user is not in (it was shared with them) is personal.
	shows(board: Board) {
		const org = board.org && this.list.some(({ id }) => id === board.org) ? board.org : PERSONAL;
		return org === this.selected;
	}
}

export const orgs = new Organizations();
