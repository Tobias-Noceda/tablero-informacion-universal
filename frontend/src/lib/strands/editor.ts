import { getContext, setContext } from 'svelte';
import type { StrandStyle } from '$types/api';

// What a strand drawn on the canvas may ask of the board around it. Flow
// provides it; the edge component only draws and forwards.
export interface StrandEditor {
	readonly canEdit: boolean;
	select(id: string): void;
	update(id: string, patch: StrandStyle): void;
	remove(id: string): void;
}

const key = Symbol('strand-editor');

export function setStrandEditor(editor: StrandEditor) {
	setContext(key, editor);
}

// Undefined outside a board, e.g. a preview: the strand is only drawn.
export function getStrandEditor(): StrandEditor | undefined {
	return getContext<StrandEditor | undefined>(key);
}
