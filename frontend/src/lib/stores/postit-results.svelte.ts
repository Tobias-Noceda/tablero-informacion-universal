import { SvelteMap } from 'svelte/reactivity';

import { nextDelay } from '$lib/helpers/auto-refresh';
import { session } from '$modules/session.svelte';
import * as postItsApi from '$services/post-it';
import type { UUID } from '$types/api';

type Result = ReturnType<typeof postItsApi.execute>;

// Who shows a card's result right now (its node, its detail drawer) and the
// timer that fetches it again.
// `due` marks one whose time came while the tab was hidden.
type Watch = { count: number; timer?: ReturnType<typeof setTimeout>; due?: boolean };

// What every card answered, shared by the card on the canvas and its detail
// drawer, so refreshing one updates both. The promises live in a plain map
// (`get` runs inside `$derived`, where state must not be written); the
// version map is what readers track.
class PostItResults {
	#requests = new Map<UUID, Result>();
	#versions = new SvelteMap<UUID, number>();
	#watches = new Map<UUID, Watch>();
	// Each card's refresh rate in seconds, null when it never fetches (a card
	// without a request); read once from its settings.
	#rates = new Map<UUID, Promise<number | null>>();

	get(id: UUID): Result {
		this.#versions.get(id);
		let request = this.#requests.get(id);
		if (!request) {
			request = postItsApi.execute(id);
			this.#requests.set(id, request);
		}
		return request;
	}

	// Runs the card again, skipping the backend's cache. Its settings may have
	// just changed, so its rate is read again too.
	refresh(id: UUID): Result {
		const request = postItsApi.execute(id, { fresh: true });
		this.#rates.delete(id);
		this.#publish(id, request);
		return request;
	}

	// Fetches the card again in the background: the current answer stays on
	// screen until a new one arrives, and stays if fetching fails.
	async revalidate(id: UUID) {
		try {
			const data = await postItsApi.execute(id);
			if (this.#watches.has(id)) this.#publish(id, Promise.resolve(data));
		} catch {
			// The next round tries again.
		}
	}

	// Keeps a card up to date while something shows it: it is fetched again
	// after a random span between its refresh rate and a minute more. Returns
	// the function that stops watching (for an $effect's cleanup).
	watch(id: UUID): () => void {
		let watch = this.#watches.get(id);
		if (watch) {
			watch.count++;
		} else {
			watch = { count: 1 };
			this.#watches.set(id, watch);
			this.#schedule(id, watch);
		}

		const current = watch;
		let stopped = false;
		return () => {
			if (stopped) return;
			stopped = true;
			if (--current.count > 0) return;
			clearTimeout(current.timer);
			if (this.#watches.get(id) === current) this.#watches.delete(id);
		};
	}

	drop(id: UUID) {
		this.#requests.delete(id);
		this.#versions.delete(id);
		this.#rates.delete(id);
		clearTimeout(this.#watches.get(id)?.timer);
		this.#watches.delete(id);
	}

	clear() {
		this.#requests.clear();
		this.#versions.clear();
		this.#rates.clear();
		for (const watch of this.#watches.values()) clearTimeout(watch.timer);
		this.#watches.clear();
	}

	#publish(id: UUID, request: Result) {
		this.#requests.set(id, request);
		this.#versions.set(id, (this.#versions.get(id) ?? 0) + 1);
	}

	#rateOf(id: UUID) {
		let rate = this.#rates.get(id);
		if (!rate) {
			rate = postItsApi
				.get_settings(id)
				.then((s) => (s.resource ? s.rate : null))
				.catch(() => {
					// Asked again on the next watch.
					this.#rates.delete(id);
					return null;
				});
			this.#rates.set(id, rate);
		}
		return rate;
	}

	async #schedule(id: UUID, watch: Watch) {
		const rate = await this.#rateOf(id);
		if (rate === null || this.#watches.get(id) !== watch) return;

		watch.timer = setTimeout(() => this.#tick(id, watch), nextDelay(rate));
	}

	async #tick(id: UUID, watch: Watch) {
		if (this.#watches.get(id) !== watch) return;
		// A hidden tab fetches nothing; it catches up when shown again.
		if (typeof document !== 'undefined' && document.hidden) {
			watch.due = true;
			return;
		}
		watch.due = false;
		await this.revalidate(id);
		this.#schedule(id, watch);
	}

	// The tab is shown again: what fell due meanwhile is fetched now.
	resume() {
		for (const [id, watch] of this.#watches) {
			if (watch.due) this.#tick(id, watch);
		}
	}
}

export const results = new PostItResults();
if (typeof document !== 'undefined') {
	document.addEventListener('visibilitychange', () => {
		if (!document.hidden) results.resume();
	});
}
// What a card answered depends on who asked.
session.onUserChange(() => results.clear());
