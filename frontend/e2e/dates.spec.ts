import { expect, test } from '@playwright/test';

import { account, BASE_URL, boards, signIn, t } from './support/stack';

// Date parameters are picked from the browser's date picker and saved as the
// card's API reads them (YYYY-MM-DD for the temperature card).
test('picks the dates of a card from a date picker', async ({ page }) => {
	const ana = await account('ana');
	const board = await boards.create(ana, 'Weather');
	await signIn(page, ana);
	await page.goto(`/boards/${board.id}`);

	await page.getByRole('button', { name: t('nodes.title') }).click();
	await page.locator('aside .temperature').dragTo(page.locator('.svelte-flow__pane'), {
		targetPosition: { x: 300, y: 200 }
	});
	const dialog = page.getByRole('dialog');
	await dialog.locator('input').first().fill('From {{min}}°C to {{max}}°C');
	const start = dialog.getByLabel('Start date');
	await expect(start).toHaveAttribute('type', 'date');
	await start.fill('2026-11-22');
	await dialog.getByLabel('End date').fill('2026-11-23');
	await dialog.getByRole('button', { name: t('nodes.modify_modal.create.action') }).click();
	await expect(dialog).toBeHidden();

	const { postits } = await boards.get(ana, board.id);
	const res = await fetch(`${BASE_URL}/api/v1/post-its/${postits[0].id}/settings`, {
		headers: { Authorization: `Bearer ${ana.token}` }
	});
	const settings = await res.json();
	expect(settings.params).toMatchObject({ $start_date: '2026-11-22', $end_date: '2026-11-23' });
});
