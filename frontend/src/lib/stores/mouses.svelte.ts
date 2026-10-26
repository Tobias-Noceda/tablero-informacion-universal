import { SvelteMap } from 'svelte/reactivity';
import { session } from '$modules/session.svelte';

export type Position = { x: number; y: number };

export type ClientData = {
	position?: Position;
	username: string;
	picture: string;
	color: string;
};

export type Mapper = (p: Position) => Position;

class MousePositions {
	#map = new SvelteMap<string, ClientData>();

	#toFlow = (p: Position) => p;

	updateMappers(flow: Mapper) {
		this.#toFlow = flow;
	}

	add(id: string, data: ClientData) {
		this.#map.set(id, data);
		return true;
	}

	remove(id: string) {
		return this.#map.delete(id);
	}

	update(id: string, position: Position) {
		const client = this.#map.get(id);
		if (!client) return false;

		// Kept in flow coordinates: the cursors are drawn inside the viewport,
		// so they follow this user's pan and zoom.
		this.#map.set(id, { ...client, position });

		return true;
	}

	convert(position: Position) {
		return this.#toFlow(position);
	}

	data(): IteratorObject<[string, ClientData]> {
		return this.#map.entries();
	}

	clear() {
		this.#map.clear();
	}
}

export const mouses = new MousePositions();
session.onUserChange(() => mouses.clear());
