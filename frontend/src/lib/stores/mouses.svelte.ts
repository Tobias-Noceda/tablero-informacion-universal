import { SvelteMap } from 'svelte/reactivity';

export type Position = { x: number; y: number };

export type ClientData = {
	position?: Position;
	username: string;
	picture: string;
};

export type Mapper = (p: Position) => Position;

class MousePositions {
	#map = new SvelteMap<string, ClientData>();

	#toFlow = (p: Position) => p;
	#toScreen = (p: Position) => p;

	updateMappers(flow: Mapper, screen: Mapper) {
		this.#toFlow = flow;
		this.#toScreen = screen;
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

		this.#map.set(id, {
			...client,
			position: this.#toScreen(position)
		});

		return true;
	}

	convert(position: Position) {
		return this.#toFlow(position);
	}

	data(): IteratorObject<ClientData> {
		return this.#map.values();
	}

	clear() {
		this.#map.clear();
	}
}

export const mouses = new MousePositions();
