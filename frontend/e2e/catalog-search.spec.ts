import { expect, test } from '@playwright/test';

import { account, boards, signIn, t } from './support/stack';

test('searches the card types in the sidebar', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Search');
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);
	await page.getByRole('button', { name: t('nodes.title') }).click();

	const sidebar = page.locator('aside');
	const search = sidebar.getByRole('searchbox', { name: t('nodes.search') });
	await expect(sidebar.getByText('Temperature')).toBeVisible();

	await search.fill('DOLAR');
	await expect(sidebar.getByText('Official Dollar')).toBeVisible();
	await expect(sidebar.getByText('Temperature')).toBeHidden();
	await expect(sidebar.getByText(t('one_off.sidebar'))).toBeVisible();

	await search.fill('no such card');
	await expect(sidebar.getByRole('status')).toHaveText(
		t('nodes.search_empty', { query: 'no such card' })
	);

	await search.press('Escape');
	await expect(search).toHaveValue('');
	await expect(sidebar.getByText('Temperature')).toBeVisible();
});
