import { session } from '$modules/session.svelte';
import { getLocale } from '$lib/paraglide/runtime';
import * as wellKnownsApi from '$services/well-knowns';
import type { WellKnown, WellKnownOutput, WellKnownParameter } from '$types/api';

// The catalogue of ready-made cards, from the backend. Loaded once per user
// (admins get a little more of each definition).
class WellKnowns {
	list = $state<WellKnown[]>([]);
	#loading: Promise<void> | undefined;
	// Bumped on reset, so a list requested for the previous user is dropped.
	#generation = 0;

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

	reset() {
		this.#generation++;
		this.#loading = undefined;
		this.list = [];
	}

	async refresh() {
		this.#loading = undefined;
		await this.ensure();
	}

	byKey(key: string | undefined) {
		return key ? this.list.find((wk) => wk.key === key) : undefined;
	}

	parameters(key: string | undefined): WellKnownParameter[] {
		return this.byKey(key)?.parameters ?? [];
	}

	outputs(key: string | undefined): WellKnownOutput[] {
		return this.byKey(key)?.outputs ?? [];
	}

	// The card's name in the current language; its key once deleted.
	name(key: string | undefined) {
		const wk = this.byKey(key);
		return wk?.[`name_${getLocale()}`] ?? key ?? '';
	}
}

export const wellKnowns = new WellKnowns();
session.onUserChange(() => wellKnowns.reset());
