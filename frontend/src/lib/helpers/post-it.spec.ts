import { describe, expect, it } from 'vitest';
import { mapPostitWithTitle, type PostItWithTitle } from './post-it';

describe('mapPostitWithTitle', () => {
	const postit: PostItWithTitle = {
		id: 'c',
		type: 'dolar_oficial',
		position: { x: 1, y: 2 },
		data: {},
		title: { text: 'Dólar', vars: false },
		color: 'white',
		border: 'dashed',
		border_width: 'thick',
		border_color: 'purple',
		width: 300,
		height: 200
	};

	// A live update maps every card again: a field left out here would be
	// lost on the canvas although the backend keeps it.
	it('keeps every style field', () => {
		const node = mapPostitWithTitle(postit);
		expect(node.data.look).toEqual({ color: 'white', border: 'dashed', border_width: 'thick', border_color: 'purple' });
		expect([node.width, node.height]).toEqual([300, 200]);
	});

	it('leaves the style fields off the node itself', () => {
		const node = mapPostitWithTitle(postit) as Record<string, unknown>;
		for (const key of ['color', 'border', 'border_width', 'border_color']) expect(key in node).toBe(false);
		expect(node.position).toEqual({ x: 1, y: 2 });
		expect((node.data as { title: unknown }).title).toEqual({ text: 'Dólar', vars: false });
	});
});
