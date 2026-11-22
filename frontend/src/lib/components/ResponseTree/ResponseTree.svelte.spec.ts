import { page } from 'vitest/browser';
import { describe, expect, it, vi } from 'vitest';
import { render } from 'vitest-browser-svelte';

import ResponseTree from './ResponseTree.svelte';

const response = {
	city: 'Buenos Aires',
	main: { temp: 21.5 },
	data: [
		{ price: 10, name: 'a' },
		{ price: 20, name: 'b' }
	]
};

describe('ResponseTree.svelte', () => {
	it('picks a value by where it sits in the response', async () => {
		const onpick = vi.fn();
		render(ResponseTree, { value: response, onpick });

		await page.getByRole('button', { name: '"Buenos Aires"' }).click();
		expect(onpick).toHaveBeenLastCalledWith(['city'], false);

		await page.getByRole('button', { name: '21.5' }).click();
		expect(onpick).toHaveBeenLastCalledWith(['main', 'temp'], false);
	});

	it('offers every element of a list, not only the clicked one', async () => {
		const onpick = vi.fn();
		render(ResponseTree, { value: response, onpick });

		// The list starts open one level deep; its items are closed.
		await page.getByText('[1]').click();
		await page.getByRole('button', { name: '20', exact: true }).click();
		expect(onpick).toHaveBeenLastCalledWith(['data', 1, 'price'], false);

		await page.getByRole('button', { name: 'all' }).first().click();
		expect(onpick).toHaveBeenLastCalledWith(['data', 1, 'price'], true);
	});

	it('picks a response that is a single value', async () => {
		const onpick = vi.fn();
		render(ResponseTree, { value: 42, onpick });

		await page.getByRole('button', { name: '42' }).click();
		expect(onpick).toHaveBeenLastCalledWith([], false);
	});
});
