import { getContext, setContext } from 'svelte';
import type { CardStyle } from '$types/api';

// Where a card ends up after a resize.
export interface CardBounds {
	x: number;
	y: number;
	width: number;
	height: number;
}

// What a card drawn on the canvas may ask of the board around it. Flow
// provides it; the node only draws and forwards.
export interface CardEditor {
	readonly canEdit: boolean;
	restyle(id: string, patch: CardStyle): void;
	// moved: a top or left handle was dragged, so the card's corner moved too.
	resized(id: string, bounds: CardBounds, moved: boolean): void;
}

const key = Symbol('card-editor');

export function setCardEditor(editor: CardEditor) {
	setContext(key, editor);
}

// Undefined outside a board, e.g. a preview: the card is only drawn.
export function getCardEditor(): CardEditor | undefined {
	return getContext<CardEditor | undefined>(key);
}
