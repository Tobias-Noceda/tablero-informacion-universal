import { session } from '$modules/session.svelte';
import { getLocale } from '$lib/paraglide/runtime';
import { m } from '$lib/paraglide/messages';
import * as wellKnownsApi from '$services/well-knowns';
import * as templatesApi from '$services/postit-templates';
import { toWellKnown } from '$lib/well-knowns/normalize';
import { ONE_OFF } from '$lib/well-knowns/one-off';
import type { UUID, WellKnown, WellKnownOutput, WellKnownParameter } from '$types/api';

// The card types: the platform's catalogue, loaded once per user (admins get
// a little more of each definition), plus the post-it templates of the board open,
// in the catalogue's shape.
class WellKnowns {
	list = $state<WellKnown[]>([]);
	// The open board's post-it templates (usable ones and those its cards use).
	templates = $state<WellKnown[]>([]);
	#loading: Promise<void> | undefined;
	// Bumped on reset, so a list requested for the previous user is dropped.
	#generation = 0;
	#board: UUID | undefined;

	// Every card type the open board knows about, for rendering.
	all = $derived([...this.list, ...this.templates]);
	// What may be dropped on the open board.
	pickable = $derived([...this.list, ...this.templates.filter((wk) => wk.usable)]);

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

	// Loads the post-it templates of board. A failure leaves none: the board
	// still works, its cards made from templates just render without labels.
	async loadBoard(board: UUID) {
		const generation = this.#generation;
		this.#board = board;
		let templates: WellKnown[] = [];
		try {
			templates = (await templatesApi.forBoard(board)).map(toWellKnown);
		} catch {
			// Shown as "no templates".
		}
		if (generation === this.#generation && this.#board === board) this.templates = templates;
	}

	// Reloads the open board's post-it templates after one changed.
	async refreshBoard() {
		if (this.#board) await this.loadBoard(this.#board);
	}

	reset() {
		this.#generation++;
		this.#loading = undefined;
		this.#board = undefined;
		this.list = [];
		this.templates = [];
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
	// placeholder for a template, whose key means nothing to anyone).
	name(key: string | undefined) {
		const wk = this.byKey(key);
		if (wk) return wk[`name_${getLocale()}`];
		if (key === ONE_OFF) return m['one_off.kind']();
		return key?.startsWith('custom_') ? m['postit_templates.deleted']() : (key ?? '');
	}
}

export const wellKnowns = new WellKnowns();
session.onUserChange(() => wellKnowns.reset());
