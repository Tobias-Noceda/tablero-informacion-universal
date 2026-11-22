import { expect, test, type Page } from '@playwright/test';

import { account, signIn, t } from './support/stack';

// Creates a board from the open creator and waits for its page, which may
// replace the page of another board.
async function createBoard(page: Page, name: string) {
	const before = page.url();
	await page.getByPlaceholder(t('board_name_placeholder')).fill(name);
	await page
		.getByRole('dialog')
		.getByRole('button', { name: t('create') })
		.click();
	await page.waitForURL(
		(url) => /\/boards\/[0-9a-f-]{36}$/.test(url.pathname) && url.href !== before
	);
}

// The second board is made from inside the first, through the top bar's
// "Create new", which used to do nothing there.
test('creates boards and keeps the cards put on them', async ({ page }) => {
	await signIn(page, await account('maker'));
	await page.goto('/');

	await page.getByRole('button', { name: t('landing.create_board') }).click();
	await createBoard(page, 'Groceries');
	const first = page.url();

	await page.getByRole('button', { name: t('nodes.title') }).click();
	await page.locator('aside .static_card').dragTo(page.locator('.svelte-flow__pane'), {
		targetPosition: { x: 300, y: 200 }
	});
	const dialog = page.getByRole('dialog');
	await dialog.locator('input').first().fill('Buy coffee');
	await dialog.getByPlaceholder('Enter text').fill('Two bags, ground');
	await dialog.getByRole('button', { name: t('nodes.modify_modal.create.action') }).click();
	await expect(page.locator('.svelte-flow__node', { hasText: 'Buy coffee' })).toBeVisible();

	await page.reload();
	await expect(page.locator('.svelte-flow__node', { hasText: 'Buy coffee' })).toBeVisible();

	// From inside a board: the board picker's "create" entry.
	await page.getByRole('button', { name: 'Groceries' }).click();
	await page.getByRole('button', { name: t('landing.create_board') }).click();
	await createBoard(page, 'Errands');
	expect(page.url()).not.toBe(first);
	await expect(page.getByRole('button', { name: 'Errands' })).toBeVisible();
	await expect(page.locator('.svelte-flow__node')).toHaveCount(0);
});
