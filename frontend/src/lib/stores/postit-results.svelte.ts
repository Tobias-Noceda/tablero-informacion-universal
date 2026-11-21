import { SvelteMap } from 'svelte/reactivity';

import { nextDelay } from '$lib/helpers/auto-refresh';
import { failureOf, type CardFailure } from '$lib/cards/failure';
import { boardPlan } from '$stores/plan.svelte';
import { session } from '$modules/session.svelte';
import * as postItsApi from '$services/post-it';
import type { UUID } from '$types/api';

type Result = ReturnType<typeof postItsApi.execute>;

// Who shows a card's result right now (its node, its detail drawer) and the
// timer that fetches it again.
// `due` marks one whose time came while the tab was hidden.
type Watch = { count: number; timer?: ReturnType<typeof setTimeout>; due?: boolean };

// When the data a card shows arrived (`at`, absent while it never answered)
// and why the last attempt to get it failed, if it did.
export type Freshness = { at?: number; failure?: CardFailure };

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
	#freshness = new SvelteMap<UUID, Freshness>();
	// Whether a card fetches anything (known once its settings are read):
	// one that never does has no update time to show.
	#polls = new SvelteMap<UUID, boolean>();

	get(id: UUID): Result {
		this.#versions.get(id);
		let request = this.#requests.get(id);
		if (!request) {
			request = postItsApi.execute(id);
			this.#requests.set(id, request);
			this.#track(id, request);
		}
		return request;
	}

	freshness(id: UUID): Freshness | undefined {
		return this.#freshness.get(id);
	}

	polls(id: UUID): boolean {
		return this.#polls.get(id) ?? false;
	}

	// Runs the card again, skipping the backend's cache: the detail's refresh
	// button, on plans that have it.
	refresh(id: UUID): Result {
		const request = postItsApi.execute(id, { fresh: true });
		this.#rates.delete(id);
		this.#publish(id, request);
		this.#track(id, request);
		return request;
	}

	// Asks for the card again after it was edited. The backend dropped its
	// cached answer with the edit, so this is a normal execution (no `fresh`,
	// which only some plans may ask for). Its rate is read again too.
	rerun(id: UUID): Result {
		const request = postItsApi.execute(id, { reload: true });
		this.#rates.delete(id);
		this.#publish(id, request);
		this.#track(id, request);
		return request;
	}

	// Fetches the card again in the background: the current answer stays on
	// screen until a new one arrives, and stays if fetching fails, marked as
	// not updated; the next round tries again.
	async revalidate(id: UUID) {
		try {
			const data = await postItsApi.execute(id);
			if (!this.#watches.has(id)) return;
			this.#publish(id, Promise.resolve(data));
			this.#arrived(id);
		} catch (err) {
			if (this.#watches.has(id)) this.#failed(id, err);
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
		this.#freshness.delete(id);
		this.#polls.delete(id);
		clearTimeout(this.#watches.get(id)?.timer);
		this.#watches.delete(id);
	}

	clear() {
		this.#requests.clear();
		this.#versions.clear();
		this.#rates.clear();
		this.#freshness.clear();
		this.#polls.clear();
		for (const watch of this.#watches.values()) clearTimeout(watch.timer);
		this.#watches.clear();
	}

	#publish(id: UUID, request: Result) {
		this.#requests.set(id, request);
		this.#versions.set(id, (this.#versions.get(id) ?? 0) + 1);
	}

	// Records how a request ended, unless a newer one replaced it meanwhile.
	// Runs after the request settles, never inside the `$derived` that asked.
	#track(id: UUID, request: Result) {
		request.then(
			() => {
				if (this.#requests.get(id) === request) this.#arrived(id);
			},
			(err) => {
				if (this.#requests.get(id) === request) this.#failed(id, err);
			}
		);
	}

	#arrived(id: UUID) {
		this.#freshness.set(id, { at: Date.now() });
	}

	// What is on screen stays, with the time it arrived.
	#failed(id: UUID, err: unknown) {
		const at = this.#freshness.get(id)?.at;
		const failure = failureOf(err);
		this.#freshness.set(id, at === undefined ? { failure } : { at, failure });
	}

	#rateOf(id: UUID) {
		let rate = this.#rates.get(id);
		if (!rate) {
			rate = postItsApi
				.get_settings(id)
				// Cards that make no request (static, image, notes: no resource)
				// never poll, not even at the plan's floor.
				.then((s) => {
					this.#polls.set(id, !!s.resource);
					return s.resource ? s.rate : null;
				})
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

		watch.timer = setTimeout(
			() => this.#tick(id, watch),
			nextDelay(rate, boardPlan.limits.min_rate)
		);
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
