import { session } from '$modules/session.svelte';
import { getLocale } from '$lib/paraglide/runtime';
import { m } from '$lib/paraglide/messages';
import * as wellKnownsApi from '$services/well-knowns';
import * as customApi from '$services/custom-well-knowns';
import { toWellKnown } from '$lib/well-knowns/normalize';
import { ONE_OFF } from '$lib/well-knowns/one-off';
import type { UUID, WellKnown, WellKnownOutput, WellKnownParameter } from '$types/api';

// The card types: the platform's catalogue, loaded once per user (admins get
// a little more of each definition), plus the custom ones of the board open,
// in the catalogue's shape.
class WellKnowns {
	list = $state<WellKnown[]>([]);
	// The open board's custom card types (usable ones and those its cards use).
	custom = $state<WellKnown[]>([]);
	#loading: Promise<void> | undefined;
	// Bumped on reset, so a list requested for the previous user is dropped.
	#generation = 0;
	#board: UUID | undefined;

	// Every card type the open board knows about, for rendering.
	all = $derived([...this.list, ...this.custom]);
	// What may be dropped on the open board.
	pickable = $derived([...this.list, ...this.custom.filter((wk) => wk.usable)]);

	ensure() {
		const generation = this.#generation;
		this.#loading ??= wellKnownsApi.list().then(
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

	// Loads the custom card types of board. A failure leaves none: the board
	// still works, its custom cards just render without labels.
	async loadBoard(board: UUID) {
		const generation = this.#generation;
		this.#board = board;
		let custom: WellKnown[] = [];
		try {
			custom = (await customApi.forBoard(board)).map(toWellKnown);
		} catch {
			// Shown as "no custom types".
		}
		if (generation === this.#generation && this.#board === board) this.custom = custom;
	}

	// Reloads the open board's custom card types after one changed.
	async refreshBoard() {
		if (this.#board) await this.loadBoard(this.#board);
	}

	reset() {
		this.#generation++;
		this.#loading = undefined;
		this.#board = undefined;
		this.list = [];
		this.custom = [];
	}

	async refresh() {
		this.#loading = undefined;
		await this.ensure();
	}

	byKey(key: string | undefined) {
		return key ? this.all.find((wk) => wk.key === key) : undefined;
	}

	// The card type's example title in the current language, shown as a hint
	// when creating a card; the card keeps whatever title it is given.
	title(key: string | undefined): string {
		return this.byKey(key)?.[`title_${getLocale()}`] ?? '';
	}

	parameters(key: string | undefined): WellKnownParameter[] {
		return this.byKey(key)?.parameters ?? [];
	}

	outputs(key: string | undefined): WellKnownOutput[] {
		return this.byKey(key)?.outputs ?? [];
	}

	// The card's name in the current language; once deleted, its key (or a
	// placeholder for a custom type, whose key means nothing to anyone).
	name(key: string | undefined) {
		const wk = this.byKey(key);
		if (wk) return wk[`name_${getLocale()}`];
		if (key === ONE_OFF) return m['one_off.kind']();
		return key?.startsWith('custom_') ? m['custom_wk.deleted']() : (key ?? '');
	}
}

export const wellKnowns = new WellKnowns();
session.onUserChange(() => wellKnowns.reset());
