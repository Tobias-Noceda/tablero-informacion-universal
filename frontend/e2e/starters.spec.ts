import { expect, test } from '@playwright/test';

import { account, boards, signIn, t } from './support/stack';

// U5: a board made from an example starts with its cards placed and linked.
test('creates a board from an example', async ({ page }) => {
	const ana = await account('ana');
	await signIn(page, ana);
	await page.goto('/');

	await page.getByRole('button', { name: t('landing.create_board') }).click();
	const dialog = page.getByRole('dialog');
	await dialog.getByRole('radio', { name: new RegExp(t('starters.trip')) }).click();
	await expect(dialog.getByPlaceholder(t('board_name_placeholder'))).toHaveValue(
		t('starters.trip')
	);
	await dialog.getByRole('button', { name: t('create') }).click();
	await page.waitForURL(/\/boards\/[0-9a-f-]{36}$/);

	await expect(page.locator('.svelte-flow__node')).toHaveCount(4);
	await expect(page.locator('.svelte-flow__edge')).toHaveCount(2);
	await expect(
		page.locator('.svelte-flow__node', { hasText: t('starters.trip_note_title') })
	).toBeVisible();

	const id = page.url().split('/').pop()!;
	const board = await boards.get(ana, id);
	expect(board.name).toBe(t('starters.trip'));
	expect(board.postits.map((p: { type: string }) => p.type).sort()).toEqual(
		['air_quality', 'dolar_oficial', 'static_card', 'temperature'].sort()
	);
});

test('keeps a name the person typed when an example is picked', async ({ page }) => {
	const ana = await account('ana');
	await signIn(page, ana);
	await page.goto('/');

	await page.getByRole('button', { name: t('landing.create_board') }).click();
	const dialog = page.getByRole('dialog');
	const name = dialog.getByPlaceholder(t('board_name_placeholder'));
	await name.fill('My money');
	await dialog.getByRole('radio', { name: new RegExp(t('starters.finance')) }).click();
	await expect(name).toHaveValue('My money');
});
