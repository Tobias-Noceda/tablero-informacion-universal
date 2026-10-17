import { SvelteMap } from 'svelte/reactivity';

import { session } from '$modules/session.svelte';
import * as postItsApi from '$services/post-it';
import type { UUID } from '$types/api';

type Result = ReturnType<typeof postItsApi.execute>;

// What every card answered, shared by the card on the canvas and its detail
// drawer, so refreshing one updates both. The promises live in a plain map
// (`get` runs inside `$derived`, where state must not be written); the
// version map is what readers track.
class PostItResults {
	#requests = new Map<UUID, Result>();
	#versions = new SvelteMap<UUID, number>();

	get(id: UUID): Result {
		this.#versions.get(id);
		let request = this.#requests.get(id);
		if (!request) {
			request = postItsApi.execute(id);
			this.#requests.set(id, request);
		}
		return request;
	}

	// Runs the card again, skipping the backend's cache.
	refresh(id: UUID): Result {
		const request = postItsApi.execute(id, { fresh: true });
		this.#requests.set(id, request);
		this.#versions.set(id, (this.#versions.get(id) ?? 0) + 1);
		return request;
	}

	drop(id: UUID) {
		this.#requests.delete(id);
		this.#versions.delete(id);
	}

	clear() {
		this.#requests.clear();
		this.#versions.clear();
	}
}

export const results = new PostItResults();
// What a card answered depends on who asked.
session.onUserChange(() => results.clear());
