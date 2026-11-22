import { expect, test } from '@playwright/test';

import { account, boards, cards, signIn, t } from './support/stack';

test('a card that cannot reach its source says so, alone', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Sources');
	// A title that shows a value makes the card read its source.
	await cards.custom(ana, board.id, 'Price: {{value}}', 'http://127.0.0.1:9/value');
	await cards.note(ana, board.id, 'Still fine');

	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	await expect(page.getByText(t('card_failure.unreachable'))).toBeVisible();
	await expect(page.locator('.svelte-flow__node', { hasText: 'Still fine' })).toBeVisible();
});

// A save the backend refuses keeps the form open with the reason, and never
// leaves a card on the canvas that was not made (here the board is deleted
// while the form is open).
test('a refused card stays in the form with the reason', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Short lived');

	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);
	await page.getByRole('button', { name: t('nodes.title') }).click();
	await page.locator('aside .static_card').dragTo(page.locator('.svelte-flow__pane'), {
		targetPosition: { x: 300, y: 200 }
	});
	const dialog = page.getByRole('dialog');
	await dialog.locator('input').first().fill('Never saved');
	await dialog.getByPlaceholder('Enter text').fill('body');

	await page.request.delete(`/api/v1/boards/${board.id}`, {
		headers: { Authorization: `Bearer ${ana.token}` }
	});
	await dialog.getByRole('button', { name: t('nodes.modify_modal.create.action') }).click();

	await expect(dialog.getByRole('alert')).toBeVisible();
	await expect(page.locator('.svelte-flow__node')).toHaveCount(0);
});
